package main

import (
	"assist/db"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func (app *App) methodCreateNote(w http.ResponseWriter, r *http.Request) error {

	params := mux.Vars(r)
	ctx := r.Context()

	squadId := params["squadId"]

	_, authLevel := app.checkAuthorization(r, "", squadId, squadAdmin|squadOwner)

	if authLevel == 0 {
		err := fmt.Errorf("Current user is not authorized to to add note to squad " + squadId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	var note db.NoteUpdate
	err := json.NewDecoder(r.Body).Decode(&note)
	if err != nil {
		err = fmt.Errorf("Failed to decode note data from the HTTP request: %w", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	id, err := app.db.CreateNote(ctx, squadId, &note)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(struct {
		ID string `json:"id"`
	}{id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return nil
}

func (app *App) methodGetNotes(w http.ResponseWriter, r *http.Request) error {
	params := mux.Vars(r)
	ctx := r.Context()

	squadId := params["squadId"]

	_, authLevel := app.checkAuthorization(r, "", squadId, squadAdmin|squadOwner|squadMember)
	if authLevel == 0 {
		err := fmt.Errorf("Current user is not authenticated to get squad " + squadId + " details")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	notes, err := app.db.GetNotes(ctx, squadId, authLevel == squadMember)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(notes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return err
}

func (app *App) methodDeleteNote(w http.ResponseWriter, r *http.Request) error {

	params := mux.Vars(r)
	ctx := r.Context()

	// authorization check
	squadId := params["squadId"]
	if _, authLevel := app.checkAuthorization(r, "", squadId, squadOwner|squadAdmin); authLevel == 0 {
		err := fmt.Errorf("Current user is not authorized to delete notes in squad " + squadId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	noteId := params["noteId"]

	err := app.db.DeleteNote(ctx, squadId, noteId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	return nil
}

func (app *App) methodUpdateNote(w http.ResponseWriter, r *http.Request) error {

	params := mux.Vars(r)
	ctx := r.Context()

	squadId := params["squadId"]
	noteId := params["noteId"]

	_, authLevel := app.checkAuthorization(r, "", squadId, squadAdmin|squadOwner)

	if authLevel == 0 {
		err := fmt.Errorf("Current user is not authorized to to add note to squad " + squadId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	var note db.NoteUpdate
	err := json.NewDecoder(r.Body).Decode(&note)
	if err != nil {
		err = fmt.Errorf("Failed to decode note data from the HTTP request: %w", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	err = app.db.UpdateNote(ctx, squadId, noteId, &note)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	return nil
}
