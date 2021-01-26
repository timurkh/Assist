package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type AuthenticatedLevel uint8

const (
	authenticatedUser AuthenticatedLevel = 1 << iota
	squadMember
	squadOwner
	admin
)

func (app *App) checkAuthorization(r *http.Request, userId *string, squadInfo *SquadInfo, requiredLevel AuthenticatedLevel) AuthenticatedLevel {

	var level AuthenticatedLevel = 0
	sd := app.getSessionData(r)
	if sd.Admin {
		level = admin
	}

	if *userId == "me" {
		*userId = app.getCurrentUserID(r)
		level = level | authenticatedUser
	}

	if squadInfo != nil {
		if requiredLevel&squadOwner != 0 && squadInfo.Owner == *userId {
			return level | squadOwner
		}

		if requiredLevel&squadMember != 0 {
			// return squadMember
		}
	}

	return level
}

// method handlers
func (app *App) methodCreateSquad(w http.ResponseWriter, r *http.Request) error {

	var squad struct{ Name string }

	err := json.NewDecoder(r.Body).Decode(&squad)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	log.Println("Creating squad " + squad.Name)

	ctx := r.Context()
	squadId, err := app.dbSquads.CreateSquad(ctx, squad.Name, app.getCurrentUserID(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(struct{ ID string }{squadId})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return err
}

func (app *App) methodGetSquads(w http.ResponseWriter, r *http.Request) error {

	ctx := r.Context()

	query := r.URL.Query()
	userId := query.Get("userId")

	// authorization check
	if app.checkAuthorization(r, &userId, nil, authenticatedUser) == 0 {
		// operation is not authorized, return error
		err := fmt.Errorf("Current user is not authorized to get squads for user %v", userId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	log.Println("Getting squads for user " + userId)

	own_squads, member_squads, other_squads, err := app.dbSquads.GetSquads(ctx, userId)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(struct {
		Own    interface{}
		Member interface{}
		Other  interface{}
	}{own_squads, member_squads, other_squads})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return err
}

func (app *App) methodDeleteSquad(w http.ResponseWriter, r *http.Request) error {

	params := mux.Vars(r)
	ctx := r.Context()

	squadId := params["id"]
	squadInfo, err := app.dbSquads.GetSquad(r.Context(), squadId)
	if err != nil {
		return err
	}

	// authorization check
	userId := "me"
	if app.checkAuthorization(r, &userId, squadInfo, squadOwner) == 0 {
		err = fmt.Errorf("Current user is not authorized to delete squad " + squadId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	log.Println("Deleting squad " + squadId)

	err = app.dbSquads.DeleteSquad(ctx, squadId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	return nil
}

func (app *App) methodGetSquad(w http.ResponseWriter, r *http.Request) error {
	params := mux.Vars(r)
	ctx := r.Context()

	squadId := params["id"]

	log.Println("Getting details for squad " + squadId)

	squadInfo, err := app.dbSquads.GetSquad(ctx, squadId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	userId := app.getCurrentUserID(r)
	if authLevel := app.checkAuthorization(r, &userId, squadInfo, squadMember|squadOwner); authLevel == 0 {
		err = fmt.Errorf("Current user is not authenticated to get squad " + squadId + " details")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(squadInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return err
}

func (app *App) methodAddMemberToSquad(w http.ResponseWriter, r *http.Request) error {

	params := mux.Vars(r)
	ctx := r.Context()

	squadId := params["squadId"]
	userId := params["userId"]

	squadInfo, err := app.dbSquads.GetSquad(ctx, squadId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	var squadUserInfo SquadUserInfo

	switch authLevel := app.checkAuthorization(r, &userId, squadInfo, authenticatedUser|squadOwner); authLevel {
	case authenticatedUser:
		squadUserInfo.Status = pendingApproveFromOwner
	case admin:
		squadUserInfo.Status = member
	case squadOwner:
		squadUserInfo.Status = pendingApproveFromMember
	default:
		err = fmt.Errorf("Current user is not authorized to to add user " + userId + " to squad " + squadId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	log.Println("Adding user " + userId + " to squad " + squadId)

	userInfo, err := app.dbUsers.GetUser(ctx, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	squadUserInfo.UserInfo = *userInfo

	err = app.dbSquads.AddMemberToSquad(ctx, squadId, userId, &squadUserInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	err = app.dbUsers.AddSquadToMember(ctx, userId, squadId, squadInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	return nil
}

func (app *App) methodDeleteMemberFromSquad(w http.ResponseWriter, r *http.Request) error {

	params := mux.Vars(r)
	ctx := r.Context()

	squadId := params["squadId"]
	userId := params["userId"]

	squadInfo, err := app.dbSquads.GetSquad(ctx, squadId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	// authorization check
	if app.checkAuthorization(r, &userId, squadInfo, authenticatedUser|squadOwner) == 0 {
		// operation is not authorized, return error
		err = fmt.Errorf("Current user is not authorized to remove user " + userId + " from squad " + squadId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	log.Println("Removing user " + userId + " from squad " + squadId)

	err = app.dbUsers.DeleteSquadFromMember(ctx, userId, squadId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	err = app.dbSquads.DeleteMemberFromSquad(ctx, squadId, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	return nil
}
