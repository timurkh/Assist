package main

import (
	"assist/db"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	gorilla_context "github.com/gorilla/context"
	"github.com/gorilla/mux"
)

func (app *App) checkAuthorizationUser(r *http.Request, userId string) (string, bool) {
	// authorization check
	sd := app.sd.getCurrentUserData(r)
	if userId == "me" {
		userId = app.sd.getCurrentUserID(r)
	} else if sd.Status == db.Admin {
		// ok, admin can do that
	} else {
		// operation is not authorized, return error
		return "", false
	}
	gorilla_context.Set(r, "AuthChecked", true)

	return userId, true
}

func (app *App) methodSetUser(w http.ResponseWriter, r *http.Request) error {
	params := mux.Vars(r)
	ctx := r.Context()

	userId := params["id"]
	userId, ok := app.checkAuthorizationUser(r, userId)
	if !ok {
		err := fmt.Errorf("Current user is not authorized to modify user %v", userId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	var user struct{ Name string }

	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	log.Printf("Updating user %v name to %v ", userId, user.Name)

	app.db.UpdateUser(ctx, userId, "DisplayName", user.Name)
	if err != nil {
		err := fmt.Errorf("Failed to update %v name: %w", userId, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
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

	// authorization check
	userId, authLevel := app.checkAuthorization(r, userId, "", myself)
	if authLevel == 0 {
		// operation is not authorized, return error
		err := fmt.Errorf("Cannot retrieve home values for user %v", userId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	userId = app.sd.getCurrentUserID(r)
	sd := app.sd.getCurrentUserData(r)

	var errs [6]error
	var squads, pendingApprove, events, eventsCount, appliedParticipants, queuesToApprove, queuesToHandle interface{}
	var wg sync.WaitGroup

	// squads
	wg.Add(1)
	go func() {
		squads, errs[0] = app.db.GetSquadsCount(ctx, userId)
		wg.Done()
	}()

	// actions
	wg.Add(1)
	go func() {
		pendingApprove, errs[1] = app.db.GetSquadsWithPendingRequests(ctx, userId, sd.Admin)
		wg.Done()
	}()

	// events
	wg.Add(1)
	go func() {
		events, errs[2] = app.db.GetUserEvents(ctx, userId, 4)
		wg.Done()
	}()

	// events count
	wg.Add(1)
	go func() {
		eventsCount, errs[3] = app.db.GetUserEventsCount(ctx, userId)
		wg.Done()
	}()

	// applied participants
	wg.Add(1)
	go func() {
		var squads []string
		squads, errs[4] = app.db.GetUserSquads(ctx, userId, "admin")
		if errs[4] == nil && len(squads) > 0 {
			appliedParticipants, errs[4] = app.db.GetEventsByStatus(ctx, squads, userId, "Applied")
		}
		wg.Done()
	}()

	// requests
	wg.Add(1)
	go func() {
		queuesToApprove, queuesToHandle, errs[5] = app.db.GetUserRequestQueues(ctx, sd.UserTags)
		wg.Done()
	}()

	wg.Wait()

	for _, e := range errs {
		if e != nil {
			http.Error(w, e.Error(), http.StatusInternalServerError)
			return e
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(
		struct {
			Squads              interface{} `json:"squads"`
			PendingApprove      interface{} `json:"pendingApprove"`
			Events              interface{} `json:"events"`
			EventsCount         interface{} `json:"eventsCount"`
			AppliedParticipants interface{} `json:"appliedParticipants"`
			QueuesToApprove     interface{} `json:"queuesToApprove"`
			QueuesToHandle      interface{} `json:"queuesToHandle"`
		}{squads, pendingApprove, events, eventsCount, appliedParticipants, queuesToApprove, queuesToHandle})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return err
}

func (app *App) methodSubscribeToNotifications(w http.ResponseWriter, r *http.Request) error {
	params := mux.Vars(r)

	userId := params["userId"]
	userId, ok := app.checkAuthorizationUser(r, userId)
	if !ok {
		// operation is not authorized, return error
		err := fmt.Errorf("Current user is not authorized to subscribe to user %v notifications", userId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	var token struct {
		Token string `json:"token"`
	}

	err := json.NewDecoder(r.Body).Decode(&token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	app.ntfs.SetUserToken(userId, token.Token)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	return nil
}

func (app *App) methodUnsubscribeFromNotifications(w http.ResponseWriter, r *http.Request) error {
	params := mux.Vars(r)

	userId := params["userId"]
	userId, ok := app.checkAuthorizationUser(r, userId)
	if !ok {
		// operation is not authorized, return error
		err := fmt.Errorf("Current user is not authorized to unsubscribe user %v from notifications", userId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	app.ntfs.DeleteUserToken(userId)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	return nil
}

func (app *App) methodGetNotifications(w http.ResponseWriter, r *http.Request) error {
	params := mux.Vars(r)

	userId := params["userId"]
	userId, ok := app.checkAuthorizationUser(r, userId)
	if !ok {
		// operation is not authorized, return error
		err := fmt.Errorf("Current user is not authorized to subscribe to user %v notifications", userId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	notifications := app.ntfs.GetNotifications(userId)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(notifications)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil
}

func (app *App) methodMarkNotificationsDelivered(w http.ResponseWriter, r *http.Request) error {
	params := mux.Vars(r)

	userId := params["userId"]
	userId, ok := app.checkAuthorizationUser(r, userId)
	if !ok {
		// operation is not authorized, return error
		err := fmt.Errorf("Current user is not authorized to subscribe to user %v notifications", userId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	app.ntfs.MarkNotificationsDelivered(userId)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	return nil
}
