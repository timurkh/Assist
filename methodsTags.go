package main

import (
	"assist/db"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func (app *App) methodCreateTag(w http.ResponseWriter, r *http.Request) error {

	params := mux.Vars(r)
	ctx := r.Context()

	squadId := params["squadId"]

	_, authLevel := app.checkAuthorization(r, "", squadId, squadAdmin|squadOwner)

	if authLevel == 0 {
		err := fmt.Errorf("Current user is not authorized to to add tag to squad " + squadId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	var tag db.Tag
	err := json.NewDecoder(r.Body).Decode(&tag)
	if err != nil {
		err = fmt.Errorf("Failed to decode tag data from the HTTP request: %w", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	err = app.db.CreateTag(ctx, squadId, &tag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	return nil
}

func (app *App) methodGetTags(w http.ResponseWriter, r *http.Request) error {
	params := mux.Vars(r)
	ctx := r.Context()

	squadId := params["squadId"]

	_, authLevel := app.checkAuthorization(r, "", squadId, squadAdmin|squadOwner)
	if authLevel == 0 {
		err := fmt.Errorf("Current user is not authenticated to get squad " + squadId + " details")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	tags, err := app.db.GetTags(ctx, squadId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(tags)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	return err
}

func (app *App) methodDeleteTag(w http.ResponseWriter, r *http.Request) error {

	params := mux.Vars(r)
	ctx := r.Context()

	// authorization check
	squadId := params["squadId"]
	if _, authLevel := app.checkAuthorization(r, "", squadId, squadOwner|squadAdmin); authLevel == 0 {
		err := fmt.Errorf("Current user is not authorized to delete tags in squad " + squadId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	tagName := params["tagName"]

	err := app.db.DeleteTag(ctx, squadId, tagName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	return nil
}
