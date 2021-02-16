package main

import (
	"assist/db"
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

	squadId := squad.Name
	ownerId := app.su.getCurrentUserID(r)

	ctx := r.Context()
	err = app.db.CreateSquad(ctx, squadId, ownerId)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	return nil
}

func (app *App) methodGetHome(w http.ResponseWriter, r *http.Request) error {

	ctx := r.Context()

	params := mux.Vars(r)
	userId := params["userId"]

	if userId != "me" {
		err := fmt.Errorf("Can retrieve home values only for myself")
		http.Error(w, err.Error(), http.StatusNotImplemented)
		return err
	}
	userId = app.su.getCurrentUserID(r)

	homeCounters, err := app.db.GetHomeCounters(ctx, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(homeCounters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return err
}

func (app *App) methodGetSquads(w http.ResponseWriter, r *http.Request) error {

	ctx := r.Context()

	params := mux.Vars(r)
	userId := params["userId"]

	// authorization check
	userId, authLevel := app.checkAuthorization(r, userId, "", authenticatedUser)
	if authLevel == 0 {
		// operation is not authorized, return error
		err := fmt.Errorf("Current user is not authorized to get squads for user %v", userId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	own_squads, other_squads, err := app.db.GetSquads(ctx, userId, authLevel&systemAdmin != 0)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
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
	userId, authLevel := app.checkAuthorization(r, userId, squadId, authenticatedUser|squadAdmin|squadOwner)

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

	err := app.db.AddMemberToSquad(ctx, userId, squadId, memberStatus)
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

func (app *App) methodCreateReplicant(w http.ResponseWriter, r *http.Request) error {

	params := mux.Vars(r)
	ctx := r.Context()

	squadId := params["squadId"]

	_, authLevel := app.checkAuthorization(r, "", squadId, squadAdmin|squadOwner)

	if authLevel == 0 {
		err := fmt.Errorf("Current user is not authorized to to add replicant to squad " + squadId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	var replicantInfo db.UserInfo
	err := json.NewDecoder(r.Body).Decode(&replicantInfo)
	if err != nil {
		err = fmt.Errorf("Failed to decode replicant data from the HTTP request: %w", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	replicantId, err := app.db.CreateReplicant(ctx, &replicantInfo, squadId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(struct{ ReplicantId string }{replicantId})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil
}

func (app *App) methodUpdateSquadMember(w http.ResponseWriter, r *http.Request) error {
	params := mux.Vars(r)
	ctx := r.Context()

	squadId := params["squadId"]
	userId := params["userId"]

	var data struct {
		Status *db.MemberStatusType
		Tag    *struct {
			Name  string
			Value string
		}
		Tags *[]interface{}
	}

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

	var ret interface{} = nil

	switch {

	case data.Status != nil:
		err = app.db.SetSquadMemberStatus(ctx, userId, squadId, *data.Status)
	case data.Tag != nil:
		tags, e := app.db.SetSquadMemberTag(ctx, userId, squadId, data.Tag.Name, data.Tag.Value)
		ret = struct {
			Tags []interface{} `json:"tags"`
		}{tags}
		err = e
	case data.Tags != nil:
		err = app.db.SetSquadMemberTags(ctx, userId, squadId, data.Tags)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(ret)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil
}

func (app *App) methodDeleteMemberFromSquad(w http.ResponseWriter, r *http.Request) error {

	params := mux.Vars(r)
	ctx := r.Context()

	squadId := params["squadId"]
	userId := params["userId"]

	// authorization check
	userId, authLevel := app.checkAuthorization(r, userId, squadId, authenticatedUser|squadOwner|squadAdmin)
	if authLevel == 0 {
		// operation is not authorized, return error
		err := fmt.Errorf("Current user is not authorized to remove user " + userId + " from squad " + squadId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	err := app.db.DeleteMemberFromSquad(ctx, userId, squadId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	return nil
}
