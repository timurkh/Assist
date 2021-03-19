package main

import (
	"assist/db"
	assist_db "assist/db"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	gorilla_context "github.com/gorilla/context"
	"github.com/gorilla/mux"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthenticatedLevel uint8

const (
	myself AuthenticatedLevel = 1 << iota
	squadMember
	squadAdmin
	squadOwner
	systemAdmin
)

func (app *App) checkAuthorization(r *http.Request, userId string, squadId string, requiredLevel AuthenticatedLevel) (_ string, level AuthenticatedLevel) {

	if app.dev {
		defer TimeTrack("checkAuthorization "+r.URL.Path, time.Now())
	}

	currentUserId := app.sd.getCurrentUserID(r)
	sd := app.sd.getCurrentUserData(r)

	if sd.Status == db.Admin {
		level = systemAdmin
	}

	if userId == "me" {
		userId = currentUserId
		if sd.Status != db.PendingApprove {
			level = level | (myself & requiredLevel)
		}
	}

	if squadId != "" {
		status, err := app.db.GetSquadMemberStatus(r.Context(), currentUserId, squadId)
		if err == nil {
			switch status {
			case assist_db.Member:
				level = level | (squadMember & requiredLevel)
			case assist_db.Admin:
				level = level | (squadAdmin & requiredLevel)
			case assist_db.Owner:
				level = level | (squadOwner & requiredLevel)
			}
		}
	}

	gorilla_context.Set(r, "AuthChecked", true)

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
	userId, authLevel := app.checkAuthorization(r, "me", "", myself)
	if authLevel == 0 {
		// operation is not authorized, return error
		err := fmt.Errorf("Current user is not authorized to create squads", userId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	ctx := r.Context()
	err = app.db.CreateSquad(ctx, squadId, userId)
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

func (app *App) methodGetUserSquads(w http.ResponseWriter, r *http.Request) error {

	ctx := r.Context()

	params := mux.Vars(r)
	userId := params["userId"]

	v := r.URL.Query()
	status := v.Get("status")

	// authorization check
	authLevel := myself
	if userId != "me" {
		authLevel = systemAdmin
	}
	userId, authLevel = app.checkAuthorization(r, userId, "", authLevel)
	if authLevel == 0 {
		// operation is not authorized, return error
		err := fmt.Errorf("Current user is not authorized to get squads for user %v", userId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	user_squads, err := app.db.GetUserSquadsMap(ctx, userId, status, authLevel&systemAdmin != 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(user_squads)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	return nil
}

func (app *App) methodGetSquads(w http.ResponseWriter, r *http.Request) error {

	ctx := r.Context()

	// authorization check
	userId, authLevel := app.checkAuthorization(r, "me", "", myself)
	if authLevel == 0 {
		// operation is not authorized, return error
		err := fmt.Errorf("Current user %v is not authorized to get squads", userId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	other_squads, err := app.db.GetSquads(ctx, userId)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(other_squads)
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

	errs := make([]error, 3)
	var ret struct {
		*db.SquadInfo
		OwnerInfo *db.UserInfo              `json:"ownerInfo"`
		Admins    []*db.SquadUserInfoRecord `json:"admins"`
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		ret.SquadInfo, errs[0] = app.db.GetSquad(r.Context(), squadId)
		if errs[0] == nil {
			ret.OwnerInfo, errs[2] = app.db.GetUser(r.Context(), ret.SquadInfo.Owner)
		}

		wg.Done()
	}()

	go func() {
		filter := map[string]string{"Status": "Admin"}
		ret.Admins, errs[1] = app.db.GetSquadMembers(r.Context(), squadId, nil, &filter)
		wg.Done()
	}()

	wg.Wait()

	for _, err := range errs {
		if err != nil {
			err = fmt.Errorf("Failed to retrieve squad %v info: %w", squadId, err)
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return err
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(ret)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return err
}

func (app *App) methodGetSquadMembers(w http.ResponseWriter, r *http.Request) (err error) {
	params := mux.Vars(r)
	ctx := r.Context()

	squadId := params["id"]
	v := r.URL.Query()
	from := v.Get("from")
	var timeFrom time.Time
	if from != "" {
		timeFrom, err = time.Parse(time.RFC3339, from)
		if err != nil {
			err = fmt.Errorf("Failed to convert from to a time struct: %w", err)
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return err
		}
	}

	filter := map[string]string{
		"Keys":   v.Get("keys"),
		"Status": v.Get("status"),
		"Tag":    v.Get("tag"),
		"Notes":  v.Get("notes"),
	}

	_, authLevel := app.checkAuthorization(r, "", squadId, squadAdmin|squadOwner)
	if authLevel == 0 {
		err := fmt.Errorf("Current user is not authenticated to get squad " + squadId + " details")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	squadMembers, err := app.db.GetSquadMembers(ctx, squadId, &timeFrom, &filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(squadMembers)
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

	var memberStatus assist_db.MemberStatusType
	userId, authLevel := app.checkAuthorization(r, userId, squadId, myself|squadAdmin|squadOwner)

	if authLevel&(squadOwner|squadAdmin|systemAdmin) != 0 {
		memberStatus = assist_db.Member
	} else if authLevel&myself != 0 {
		memberStatus = assist_db.PendingApprove
	} else {
		err := fmt.Errorf("Current user is not authorized to to add user " + userId + " to squad " + squadId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	squadInfo, err := app.db.AddMemberToSquad(ctx, userId, squadId, memberStatus)
	if err != nil {
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

	var replicantInfo assist_db.UserInfo
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
		Status *assist_db.MemberStatusType `json:"status"`
		Notes  *map[string]string          `json:"notes"`
	}

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	// authorization check
	userId, authLevel := app.checkAuthorization(r, userId, squadId, squadAdmin|squadOwner)
	if authLevel == 0 {
		// operation is not authorized, return error
		err := fmt.Errorf("Current user is not authorized to change user " + userId + " status in squad " + squadId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	switch {
	case data.Status != nil:
		err = app.db.SetSquadMemberStatus(ctx, userId, squadId, *data.Status)
	case data.Notes != nil:
		err = app.db.SetSquadMemberNotes(ctx, userId, squadId, data.Notes)

	}

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
	userId, authLevel := app.checkAuthorization(r, userId, squadId, myself|squadOwner|squadAdmin)
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
