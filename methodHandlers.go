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

func (app *App) checkAuthorization(r *http.Request, userId_ string, squadId string, requiredLevel AuthenticatedLevel) (userId string, squadInfo *SquadInfo, level AuthenticatedLevel) {

	sd := app.su.getSessionData(r)
	if sd.Admin {
		level = admin
	}

	userId = userId_
	if userId_ == "me" {
		userId = app.su.getCurrentUserID(r)
		level = level | authenticatedUser
	}

	if requiredLevel&squadOwner != 0 {
		var err error
		squadInfo, err = app.db.GetSquad(r.Context(), squadId)
		if err != nil {
			log.Println("Error while checking user authorization: " + err.Error())
		} else if squadInfo.Owner == userId {
			return userId, squadInfo, level | squadOwner // exit here to avoid extra query to check if user is member
		}
	}

	if requiredLevel&squadMember != 0 {
		err := app.db.CheckIfUserIsSquadMember(r.Context(), userId, squadId)
		if err == nil {
			level = level | squadMember
		}
	}

	return userId, squadInfo, level
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
	squadId, err := app.db.CreateSquad(ctx, squad.Name, app.su.getCurrentUserID(r))
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
	userId, _, authLevel := app.checkAuthorization(r, userId, "", authenticatedUser)
	if authLevel == 0 {
		// operation is not authorized, return error
		err := fmt.Errorf("Current user is not authorized to get squads for user %v", userId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	log.Println("Getting squads for user " + userId)

	own_squads, member_squads, other_squads, err := app.db.GetSquads(ctx, userId)

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

	// authorization check
	squadId := params["id"]
	if _, _, authLevel := app.checkAuthorization(r, "me", squadId, squadOwner); authLevel == 0 {
		err := fmt.Errorf("Current user is not authorized to delete squad " + squadId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	log.Println("Deleting squad " + squadId)

	err := app.db.DeleteSquad(ctx, squadId)
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

	squadId := params["id"]

	log.Println("Getting details for squad " + squadId)

	_, squadInfo, authLevel := app.checkAuthorization(r, "me", squadId, squadMember|squadOwner)
	if authLevel == 0 {
		err := fmt.Errorf("Current user is not authenticated to get squad " + squadId + " details")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(squadInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return err
}

func (app *App) methodGetSquadMembers(w http.ResponseWriter, r *http.Request) error {
	params := mux.Vars(r)
	ctx := r.Context()

	squadId := params["id"]

	_, squadInfo, authLevel := app.checkAuthorization(r, "me", squadId, squadMember|squadOwner)
	if authLevel == 0 {
		err := fmt.Errorf("Current user is not authenticated to get squad " + squadId + " details")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	log.Println("Getting members of the squad " + squadId)
	squadMembers, err := app.db.GetSquadMembers(ctx, squadId)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	log.Println("Getting info about owner of the squad " + squadId)
	squadOwner, err := app.db.GetUser(ctx, squadInfo.Owner)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(struct {
		Owner   interface{}
		Members interface{}
	}{squadOwner, squadMembers})
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

	var squadUserInfo SquadUserInfo
	userId, squadInfo, authLevel := app.checkAuthorization(r, userId, squadId, authenticatedUser|squadOwner)

	switch authLevel {
	case authenticatedUser:
		squadUserInfo.Status = pendingApproveFromOwner
	case admin:
		squadUserInfo.Status = member
	case squadOwner:
		squadUserInfo.Status = pendingApproveFromMember
	default:
		err := fmt.Errorf("Current user is not authorized to to add user " + userId + " to squad " + squadId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	log.Println("Adding user " + userId + " to squad " + squadId)

	userInfo, err := app.db.GetUser(ctx, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}
	squadUserInfo.UserInfo = *userInfo

	err = app.db.AddMemberToSquad(ctx, squadId, userId, &squadUserInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	err = app.db.AddSquadToMember(ctx, userId, squadId, squadInfo)
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

	// authorization check
	userId, _, authLevel := app.checkAuthorization(r, userId, squadId, authenticatedUser|squadOwner)
	if authLevel == 0 {
		// operation is not authorized, return error
		err := fmt.Errorf("Current user is not authorized to remove user " + userId + " from squad " + squadId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	log.Println("Removing user " + userId + " from squad " + squadId)

	err := app.db.DeleteSquadFromMember(ctx, userId, squadId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	err = app.db.DeleteMemberFromSquad(ctx, squadId, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	return nil
}
