package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	assist_db "assist/db"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	gorilla_context "github.com/gorilla/context"
)

type SessionUtil struct {
	authClient *auth.Client
	db         *assist_db.FirestoreDB
	dev        bool
}

func TimeTrack(name string, start time.Time) {
	elapsed := time.Since(start)

	log.Println(fmt.Sprintf("%s took %s", name, elapsed))
}

// init firebase auth
func initSessionUtil(fireapp *firebase.App, db *assist_db.FirestoreDB, dev bool) *SessionUtil {
	ctx := context.Background()

	authClient, err := fireapp.Auth(ctx)
	if err != nil {
		log.Fatalf("firebase.Auth: %v", err)
	}
	mdlwr := SessionUtil{
		authClient, db, dev}

	return &mdlwr
}

func (su *SessionUtil) sessionLogin(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	// Get the tokens sent by the client
	idToken := r.FormValue("idToken")

	// Decode the IDToken
	decoded, err := su.authClient.VerifyIDToken(ctx, idToken)
	if err != nil {
		str := err.Error()
		http.Error(w, str, http.StatusUnauthorized)
		return err
	}
	time_now := time.Now().Unix()
	claimed_auth_time := int64(decoded.Claims["auth_time"].(float64))
	// Return error if the sign-in is older than 5 minutes.
	if time_now-claimed_auth_time > 5*60 {
		err = errors.New(fmt.Sprintf("Recent sign-in required, claimed_auth_time=%v, time_now=%v", time_now, claimed_auth_time))
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	// Set session expiration to 5 days.
	expiresIn := time.Hour * 24 * 5

	// Create the session cookie. This will also verify the ID token in the process.
	// The session cookie will have the same claims as the ID token.
	// To only allow session cookie setting on recent sign-in, auth_time in ID token
	// can be checked to ensure user was recently signed in before creating a session cookie.
	cookie, err := su.authClient.SessionCookie(ctx, idToken, expiresIn)
	if err != nil {
		err = errors.New("Failed to create a session cookie: " + err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	// Set cookie policy for session cookie.
	http.SetCookie(w, &http.Cookie{
		Name:     "firebaseSession",
		Value:    cookie,
		MaxAge:   int(expiresIn.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   !su.dev,
	})
	w.Write([]byte(`{"status": "success"}`))

	// Check if user exists in DB, add record otherwise
	userId := decoded.UID
	userRecord, err := su.authClient.GetUser(ctx, userId)
	if err != nil {
		return fmt.Errorf("Failed to get user record: %w", err)
	}

	err = su.db.UpdateUserInfoFromFirebase(ctx, userRecord)
	if err != nil {
		return fmt.Errorf("Failed to update user info: %w", err)
	}

	http.Redirect(w, r, "/home", http.StatusFound)
	return nil
}

func (su *SessionUtil) sessionLogout(w http.ResponseWriter, r *http.Request) error {
	http.SetCookie(w, &http.Cookie{
		Name:     "firebaseSession",
		Value:    "",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   0,
	})

	http.Redirect(w, r, "/login", http.StatusFound)
	return nil
}

func (su *SessionUtil) isSessionValid(w http.ResponseWriter, r *http.Request) bool {
	if su.dev {
		defer TimeTrack("isSessionValid "+r.URL.Path, time.Now())
	}
	switch r.URL.Path {
	case "/sessionLogin":
	case "/sessionLogout":
	case "/login":
	default:
		// Get the ID token sent by the client
		cookie, err := r.Cookie("firebaseSession")
		if err != nil {
			if r.URL.Path != "/about" {
				log.Print("Session cookie is unavailable. Force user to login.")
				http.Redirect(w, r, "/login", http.StatusFound)
				return false
			}
		} else {

			// Check refoked token is quite expensive, do not do it for now
			//decoded, err := su.authClient.VerifySessionCookieAndCheckRevoked(r.Context(), cookie.Value)
			decoded, err := su.authClient.VerifySessionCookie(r.Context(), cookie.Value)
			if err != nil {
				if r.URL.Path != "/about" {
					log.Print("Session cookie is invalid. Force user to login.")
					http.Redirect(w, r, "/login", http.StatusFound)
					return false
				}
			} else {
				gorilla_context.Set(r, "SessionToken", decoded)
				sd, err := su.db.GetUserData(r.Context(), decoded.UID)
				if err != nil {
					if r.URL.Path != "/about" {
						err = errors.New("Failed to get user details: " + err.Error())
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return false
					}
				} else {
					gorilla_context.Set(r, "SessionData", sd)
					if sd.Status == assist_db.PendingApprove {
						log.Print("User is pending approve. Redirect to /userinfo")
						http.Redirect(w, r, "/userinfo", http.StatusFound)
						return false
					}
				}
			}
		}
	}
	return true
}

func (su *SessionUtil) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if su.dev {
			defer TimeTrack("Processing "+r.URL.Path, time.Now())
		}
		if su.isSessionValid(w, r) {
			next.ServeHTTP(w, r)
		}
	})
}

func (su *SessionUtil) getCurrentUserID(r *http.Request) string {
	sessionToken := gorilla_context.Get(r, "SessionToken")

	if sessionToken != nil {
		return sessionToken.(*auth.Token).UID
	} else {
		return ""
	}
}

func (su *SessionUtil) getCurrentUserData(r *http.Request) *assist_db.UserData {
	sessionData := gorilla_context.Get(r, "SessionData")

	return sessionData.(*assist_db.UserData)
}

func (su *SessionUtil) getCurrentUserRecord(r *http.Request) (*auth.UserRecord, error) {
	u, err := su.authClient.GetUser(r.Context(), su.getCurrentUserID(r))

	return u, err
}
