package main

import (
	"assist/db"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthenticatedLevel uint8

const (
	authenticatedUser AuthenticatedLevel = 1 << iota
	squadMember
	squadAdmin
	squadOwner
	systemAdmin
)

func (app *App) checkAuthorization(r *http.Request, userId string, squadId string, requiredLevel AuthenticatedLevel) (_ string, level AuthenticatedLevel) {

	sd := app.su.getSessionData(r)
	if sd.Admin {
		level = systemAdmin
	}

	currentUserId := app.su.getCurrentUserID(r)
	if userId == "me" {
		userId = currentUserId
		level = level | (authenticatedUser & requiredLevel)
	}

	if squadId != "" {
		status, err := app.db.GetSquadMemberStatus(r.Context(), currentUserId, squadId)
		if err == nil {
			switch status {
			case db.Member:
				level = level | (squadMember & requiredLevel)
			case db.Admin:
				level = level | (squadAdmin & requiredLevel)
			case db.Owner:
				level = level | (squadOwner & requiredLevel)
			}
		}
	}

	return userId, level
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
	squadId := squad.Name

	ownerId := app.su.getCurrentUserID(r)

	squadInfo := &db.SquadInfo{
		ownerId,
		1,
	}

	ctx := r.Context()
	err = app.db.CreateSquad(ctx, squadId, squadInfo)
	if err != nil {
		st, ok := status.FromError(err)
		err = fmt.Errorf("Failed to create squad %v: %w", squadId, err)
		if ok && st.Code() == codes.AlreadyExists {
			http.Error(w, err.Error(), http.StatusConflict)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return err
	}

	return app.addUserToSquad(ctx, ownerId, squadId, db.Owner)
}

func (app *App) methodGetSquads(w http.ResponseWriter, r *http.Request) error {

	ctx := r.Context()

	query := r.URL.Query()
	userId := query.Get("userId")

	// authorization check
	userId, authLevel := app.checkAuthorization(r, userId, "", authenticatedUser)
	if authLevel == 0 {
		// operation is not authorized, return error
		err := fmt.Errorf("Current user is not authorized to get squads for user %v", userId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	log.Println("Getting squads for user " + userId)

	own_squads, other_squads, err := app.db.GetSquads(ctx, userId)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	if authLevel&systemAdmin != 0 {
		uberSquadInfo, err := app.db.GetSquad(ctx, db.ALL_USERS_SQUAD)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return err
		}

		var uberSquad db.MemberSquadInfoRecord
		uberSquad.ID = db.ALL_USERS_SQUAD
		uberSquad.SquadInfo = *uberSquadInfo
		uberSquad.Status = db.Owner
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return err
		}

		own_squads = append([]*db.MemberSquadInfoRecord{&uberSquad}, own_squads...)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(struct {
		Own   interface{}
		Other interface{}
	}{own_squads, other_squads})
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
	if _, authLevel := app.checkAuthorization(r, "", squadId, squadOwner); authLevel == 0 {
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

	_, authLevel := app.checkAuthorization(r, "", squadId, squadMember|squadOwner)
	if authLevel == 0 {
		err := fmt.Errorf("Current user is not authenticated to get squad " + squadId + " details")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	squadInfo, err := app.db.GetSquad(r.Context(), squadId)
	if err != nil {
		err = fmt.Errorf("Failed to retrieve squad %v info: %w", squadId, err.Error())
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

func (app *App) methodGetSquadMembers(w http.ResponseWriter, r *http.Request) error {
	params := mux.Vars(r)
	ctx := r.Context()

	squadId := params["id"]

	_, authLevel := app.checkAuthorization(r, "", squadId, squadAdmin|squadOwner)
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

	var memberStatus db.MemberStatusType
	userId, authLevel := app.checkAuthorization(r, userId, squadId, authenticatedUser|squadOwner)

	if authLevel&(squadOwner|squadAdmin|systemAdmin) != 0 {
		memberStatus = db.Member
	} else if authLevel&authenticatedUser != 0 {
		memberStatus = db.PendingApprove
	} else {
		err := fmt.Errorf("Current user is not authorized to to add user " + userId + " to squad " + squadId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	err := app.addUserToSquad(ctx, userId, squadId, memberStatus)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(struct{ Status db.MemberStatusType }{memberStatus})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil
}

func (app *App) addUserToSquad(ctx context.Context, userId string, squadId string, memberStatus db.MemberStatusType) error {
	log.Println("Adding user " + userId + " to squad " + squadId)

	userInfo, err := app.db.GetUser(ctx, userId)
	if err != nil {
		return err
	}

	squadUserInfo := &db.SquadUserInfo{
		UserInfo: *userInfo,
		Status:   memberStatus,
	}

	err = app.db.AddMemberToSquad(ctx, squadId, userId, squadUserInfo)
	if err != nil {
		return err
	}

	squadInfo, err := app.db.GetSquad(ctx, squadId)
	if err != nil {
		return err
	}

	memberSquadInfo := &db.MemberSquadInfo{
		SquadInfo: *squadInfo,
		Status:    memberStatus,
	}

	err = app.db.AddSquadToMember(ctx, userId, squadId, memberSquadInfo)
	if err != nil {
		return err
	}

	return nil
}

func (app *App) methodChangeSquadMemberStatus(w http.ResponseWriter, r *http.Request) error {
	params := mux.Vars(r)
	ctx := r.Context()

	squadId := params["squadId"]
	userId := params["userId"]

	var data struct{ Status db.MemberStatusType }

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	// authorization check
	userId, authLevel := app.checkAuthorization(r, userId, squadId, squadOwner)
	if authLevel == 0 {
		// operation is not authorized, return error
		err := fmt.Errorf("Current user is not authorized to change user " + userId + " status in squad " + squadId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	err = app.db.SetSquadMemberStatus(ctx, userId, squadId, data.Status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	userId, authLevel := app.checkAuthorization(r, userId, squadId, authenticatedUser|squadOwner)
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
