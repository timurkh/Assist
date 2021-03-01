package main

import (
	assist_db "assist/db"
	"net/http"

	"firebase.google.com/go/auth"
)

type SessionDataGetter interface {
	getCurrentUserData(r *http.Request) *assist_db.UserData
	getCurrentUserID(r *http.Request) string
	getCurrentUserRecord(r *http.Request) (*auth.UserRecord, error)
}

type SessionMiddleware interface {
	authMiddleware(next http.Handler) http.Handler
	sessionLogin(w http.ResponseWriter, r *http.Request) error
	sessionLogout(w http.ResponseWriter, r *http.Request) error
}
