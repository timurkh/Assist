package main

import (
	"log"
	"net/http"
	"os"
	"time"

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
	r := mux.NewRouter().StrictSlash(true)

	r.Use(app.su.authMiddleware)

	// auth handlers
	r.Methods("POST").Path("/sessionLogin").Handler(appHandler(app.su.sessionLogin))
	r.Methods("POST").Path("/sessionLogout").Handler(appHandler(app.su.sessionLogout))

	// tab handlers
	r.Methods("GET").Path("/home").Handler(appHandler(app.homeHandler))
	r.Methods("GET").Path("/login").Handler(appHandler(app.loginHandler))
	r.Methods("GET").Path("/userinfo").Handler(appHandler(app.userinfoHandler))
	r.Methods("GET").Path("/squads").Handler(appHandler(app.squadsHandler))
	r.Methods("GET").Path("/squad").Handler(appHandler(app.squadHandler))
	r.Methods("GET").Path("/events").Handler(appHandler(app.eventsHandler))
	r.Methods("GET").Path("/about").Handler(appHandler(app.aboutHandler))

	r.Handle("/", http.RedirectHandler("/home", http.StatusFound))

	// methods
	r.Methods("PUT").Path("/methods/squads/{squadId}/members/{userId}").Handler(appHandler(app.methodAddMemberToSquad))
	r.Methods("DELETE").Path("/methods/squads/{squadId}/members/{userId}").Handler(appHandler(app.methodDeleteMemberFromSquad))
	r.Methods("POST").Path("/methods/squads").Handler(appHandler(app.methodCreateSquad))
	r.Methods("GET").Path("/methods/squads").Handler(appHandler(app.methodGetSquads))
	r.Methods("DELETE").Path("/methods/squads/{id}").Handler(appHandler(app.methodDeleteSquad))
	r.Methods("GET").Path("/methods/squads/{id}").Handler(appHandler(app.methodGetSquad))
	r.Methods("GET").Path("/methods/squads/{id}/members").Handler(appHandler(app.methodGetSquadMembers))

	// setup logging
	http.Handle("/", handlers.CombinedLoggingHandler(os.Stdout, r))

	// server js files & turn off caching
	http.Handle("/static/", NoCache(http.StripPrefix("/static/", http.FileServer(http.Dir("./static")))))

	// get rid of favicon errors in logs
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/favicon.ico")
	})
}
