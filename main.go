package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"context"

	"firebase.google.com/go/auth"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var (
	firebaseTmpl = parseTemplate("firebase.html")
	loginTmpl    = parseTemplate("login.html")
	homeTmpl     = parseBodyTemplate("home.html")
	userinfoTmpl = parseBodyTemplate("userinfo.html")
)

// trick to conver my functions to http.Handler
type appHandler func(http.ResponseWriter, *http.Request) error

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil {
		log.Println("appHandler error:" + e.Error())
	}
}

func initApp() (*App, error) {
	app := App{
		logWriter: os.Stderr,
	}

	ctx := context.Background()

	// init fireapp
	fireapp, err := firebase.NewApp(ctx, nil)
	if err != nil {
		log.Fatalf("firebase.NewApp: %v", err)
	}

	// init firestore
	client, err := fireapp.Firestore(ctx)
	if err != nil {
		log.Fatalf("fireapp.Firestore: %v", err)
	}

	err = client.RunTransaction(ctx, func(ctx context.Context, t *firestore.Transaction) error {
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("firestoredb: could not connect: %v", err)
	}

	app.db = newFirestoreDB(client)

	// init firebase auth
	app.authClient, err = fireapp.Auth(ctx)
	if err != nil {
		log.Fatalf("firebase.Auth: %v", err)
	}

	return &app, nil
}

type App struct {
	logWriter  io.Writer
	db         *firestoreDB
	authClient *auth.Client
}

func (app *App) homeHandler(w http.ResponseWriter, r *http.Request) error {

	u, _ := app.getCurrentUserInfo(r)
	return homeTmpl.Execute(app, w, r, struct {
		Session *SessionData
		Data    string
	}{
		app.getSessionData(r),
		fmt.Sprintf("%+v<br>%+v", u, u.ProviderUserInfo[0])})
}

func (app *App) userinfoHandler(w http.ResponseWriter, r *http.Request) error {

	ctx := context.Background()
	users := app.db.getUsersDatabase()

	sessionData := app.getSessionData(r)
	user, _ := users.GetUserDetails(ctx, sessionData.UID)

	return userinfoTmpl.Execute(app, w, r, struct {
		Session *SessionData
		Data    *UserDetails
	}{sessionData, user})
}

func (app *App) loginHandler(w http.ResponseWriter, r *http.Request) error {
	return loginTmpl.Execute(app, w, r, nil)
}

func (app *App) registerHandlers() {
	r := mux.NewRouter().StrictSlash(true)

	r.Use(app.authMiddleware)

	r.Methods("POST").Path("/sessionLogin").Handler(appHandler(app.sessionLogin))
	r.Methods("POST").Path("/sessionLogout").Handler(appHandler(app.sessionLogout))

	r.Methods("GET").Path("/home").Handler(appHandler(app.homeHandler))
	r.Methods("GET").Path("/login").Handler(appHandler(app.loginHandler))
	r.Methods("GET").Path("/userinfo").Handler(appHandler(app.userinfoHandler))

	r.Handle("/", http.RedirectHandler("/home", http.StatusFound))

	http.Handle("/", handlers.CombinedLoggingHandler(app.logWriter, r))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/favicon.ico")
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	app, err := initApp()
	if err != nil {
		log.Fatalf("Failed to init app: %v", err)
	}

	app.registerHandlers()

	log.Printf("Listening on localhost: %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
