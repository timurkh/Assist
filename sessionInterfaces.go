package main

import (
	"net/http"

	"firebase.google.com/go/auth"
)

type SessionData struct {
	*auth.UserRecord
	ContactInfoIssues    bool
	DisplayNameNotUnique bool
	PendingApproval      bool
	Role                 string
	Admin                bool
}

type SessionDataGetter interface {
	getSessionData(r *http.Request) *SessionData
	getCurrentUserID(r *http.Request) string
}

type SessionMiddleware interface {
	authMiddleware(next http.Handler) http.Handler
	sessionLogin(w http.ResponseWriter, r *http.Request) error
	sessionLogout(w http.ResponseWriter, r *http.Request) error
}
