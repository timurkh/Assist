package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func (app *App) methodSetUser(w http.ResponseWriter, r *http.Request) error {
	params := mux.Vars(r)
	ctx := r.Context()

	userId := params["id"]
	// authorization check
	sd := app.su.getSessionData(r)
	if userId == "me" {
		userId = app.su.getCurrentUserID(r)
	} else if sd.Admin {
		// ok, admin can do that
	} else {
		// operation is not authorized, return error
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

	log.Println("Updating user %v name to %v ", userId, user.Name)

	app.db.UpdateUser(ctx, userId, "name", user.Name)
	if err != nil {
		err := fmt.Errorf("Failed to update %v name: %w", userId, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	return nil
}
