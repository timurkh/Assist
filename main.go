package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"context"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
)

func initApp() (*App, error) {
	//
	app := App{
		logWriter: os.Stderr,
	}

	// init logs
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

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

	db := newFirestoreDB(client)
	app.dbUsers = db
	app.dbSquads = db

	// init firebase auth
	authClient, err := fireapp.Auth(ctx)
	if err != nil {
		log.Fatalf("firebase.Auth: %v", err)
	}
	app.su = initSessionUtil(authClient, db)

	return &app, nil
}

type App struct {
	logWriter io.Writer
	dbUsers   UsersDatabase
	dbSquads  SquadsDatabase
	su        *SessionUtil
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
