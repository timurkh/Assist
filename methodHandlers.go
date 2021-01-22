package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

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
	if userId == "me" {
		userId = app.getCurrentUserID(r)
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
	log.Println("Deleting squad " + squadId)

	err := app.dbSquads.DeleteSquad(ctx, squadId)
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

	squad, err := app.dbSquads.GetSquad(ctx, squadId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(squad)
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
	if userId == "me" {
		userId = app.getCurrentUserID(r)
	}
	log.Println("Adding user " + userId + " to squad " + squadId)

	userInfo, err := app.dbUsers.GetUser(ctx, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	err = app.dbSquads.AddMemberToSquad(ctx, squadId, userId, userInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	squadInfo, err := app.dbSquads.GetSquad(ctx, squadId)
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
	if userId == "me" {
		userId = app.getCurrentUserID(r)
	}
	log.Println("Removing user " + userId + " from squad " + squadId)

	err := app.dbUsers.DeleteSquadFromMember(ctx, userId, squadId)
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
