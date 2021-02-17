package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/csrf"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var etagHeaders = []string{
	"ETag",
	"If-Modified-Since",
	"If-Match",
	"If-None-Match",
	"If-Range",
	"If-Unmodified-Since",
}

// no-cache handlers for JS scripts
var noCacheHeaders = map[string]string{
	"Expires":         time.Unix(0, 0).Format(time.RFC1123),
	"Cache-Control":   "no-cache, private, max-age=0",
	"Pragma":          "no-cache",
	"X-Accel-Expires": "0",
}

func NoCache(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// Delete any ETag headers that may have been set
		for _, v := range etagHeaders {
			if r.Header.Get(v) != "" {
				r.Header.Del(v)
			}
		}

		// Set our NoCache headers
		for k, v := range noCacheHeaders {
			w.Header().Set(k, v)
		}

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// trick to conver my functions to http.Handler
type appHandler func(http.ResponseWriter, *http.Request) error

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil {
		log.Println("appHandler error:" + e.Error())
	}
}

func (app *App) registerHandlers() {
	// serve js files & turn off caching
	if app.dev {
		http.Handle("/static/", NoCache(http.StripPrefix("/static/", http.FileServer(http.Dir("./static")))))
	} else {
		http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	}

	// get rid of favicon errors in logs
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/favicon.ico")
	})

	r := mux.NewRouter().StrictSlash(true)
	CSRF := csrf.Protect(
		[]byte("dG3d563vyukewv%Yetrsbvsfd%WYfvs!"),
		csrf.SameSite(csrf.SameSiteStrictMode),
		csrf.Secure(!app.dev),
		csrf.HttpOnly(true),
		csrf.Path("/"),
	)

	r.Use(CSRF)
	r.Use(app.su.authMiddleware)

	// Methods
	rm := r.PathPrefix("/methods").Subrouter()

	rm.Methods("POST").Path("/squads/{squadId}/members").Handler(appHandler(app.methodCreateReplicant))
	rm.Methods("POST").Path("/squads/{squadId}/members/{userId}").Handler(appHandler(app.methodAddMemberToSquad))
	rm.Methods("PATCH").Path("/squads/{squadId}/members/{userId}").Handler(appHandler(app.methodUpdateSquadMember))
	rm.Methods("DELETE").Path("/squads/{squadId}/members/{userId}").Handler(appHandler(app.methodDeleteMemberFromSquad))
	rm.Methods("POST").Path("/squads").Handler(appHandler(app.methodCreateSquad))
	rm.Methods("DELETE").Path("/squads/{id}").Handler(appHandler(app.methodDeleteSquad))
	rm.Methods("GET").Path("/squads/{id}").Handler(appHandler(app.methodGetSquad))
	rm.Methods("GET").Path("/squads/{id}/members").Handler(appHandler(app.methodGetSquadMembers))

	rm.Methods("PUT").Path("/users/{id}").Handler(appHandler(app.methodSetUser))
	rm.Methods("GET").Path("/users/{userId}/squads").Handler(appHandler(app.methodGetSquads))
	rm.Methods("GET").Path("/users/{userId}/home").Handler(appHandler(app.methodGetHome))

	// tags
	rm.Methods("POST").Path("/squads/{squadId}/tags").Handler(appHandler(app.methodCreateTag))
	rm.Methods("GET").Path("/squads/{squadId}/tags").Handler(appHandler(app.methodGetTags))
	rm.Methods("DELETE").Path("/squads/{squadId}/tags/{tagName}").Handler(appHandler(app.methodDeleteTag))
	rm.Methods("POST").Path("/squads/{squadId}/members/{userId}/tags").Handler(appHandler(app.methodSetMemberTag))
	rm.Methods("DELETE").Path("/squads/{squadId}/members/{userId}/tags/{tagName}").Handler(appHandler(app.methodDeleteMemberTag))
	rm.Methods("DELETE").Path("/squads/{squadId}/members/{userId}/tags/{tagName}/{tagValue}").Handler(appHandler(app.methodDeleteMemberTag))

	// notes
	rm.Methods("PUT").Path("/squads/{squadId}/notes/{noteId}").Handler(appHandler(app.methodUpdateNote))
	rm.Methods("POST").Path("/squads/{squadId}/notes").Handler(appHandler(app.methodCreateNote))
	rm.Methods("GET").Path("/squads/{squadId}/notes").Handler(appHandler(app.methodGetNotes))
	rm.Methods("DELETE").Path("/squads/{squadId}/notes/{noteId}").Handler(appHandler(app.methodDeleteNote))

	// auth handlers
	r.Methods("POST").Path("/sessionLogin").Handler(appHandler(app.su.sessionLogin))
	r.Methods("POST").Path("/sessionLogout").Handler(appHandler(app.su.sessionLogout))

	// tab handlers
	r.Methods("GET").Path("/home").Handler(appHandler(app.homeHandler))
	r.Methods("GET").Path("/login").Handler(appHandler(app.loginHandler))
	r.Methods("GET").Path("/userinfo").Handler(appHandler(app.userinfoHandler))
	r.Methods("GET").Path("/squads/{squadId}/members").Handler(appHandler(app.squadMembersHandler))
	r.Methods("GET").Path("/squads/{squadId}").Handler(appHandler(app.squadDetailsHandler))
	r.Methods("GET").Path("/squads").Handler(appHandler(app.squadsHandler))
	r.Methods("GET").Path("/events").Handler(appHandler(app.eventsHandler))
	r.Methods("GET").Path("/about").Handler(appHandler(app.aboutHandler))

	r.Handle("/", http.RedirectHandler("/home", http.StatusFound))
	http.Handle("/", handlers.CombinedLoggingHandler(os.Stdout, r))

}
