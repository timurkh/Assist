package main

import (
	"context"
	"encoding/json"
	"net/http"
)

// method handlers
func (app *App) methodGetSquadHandler(w http.ResponseWriter, r *http.Request) error {

	var squad struct{ Name string }

	err := json.NewDecoder(r.Body).Decode(&squad)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	ctx := context.Background()
	squadId, err := app.db.AddSquad(ctx, squad.Name, getCurrentUserID(r))
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

func (app *App) methodPostSquadHandler(w http.ResponseWriter, r *http.Request) error {

	ctx := context.Background()
	squadId, err := app.db.GetSquads(ctx, getCurrentUserID)
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
