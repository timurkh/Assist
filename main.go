package main

import (
	"assist/db"
	"io"
	"log"
	"net/http"
	"os"

	"context"

	firebase "firebase.google.com/go"
)

func initApp(dev bool) (*App, error) {
	//
	app := App{
		logWriter: os.Stderr,
		dev:       dev,
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
	app.db, err = db.NewFirestoreDB(fireapp)
	if err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}

	// init firebase auth
	app.su = initSessionUtil(fireapp, app.db, dev)

	return &app, nil
}

type App struct {
	logWriter io.Writer
	db        *db.FirestoreDB
	su        *SessionUtil
	dev       bool
}

func main() {
	args := os.Args[1:]

	dev := true
	if len(args) == 1 && args[0] == "--dev" {
		log.Println("Running in DEBUG mode")
		dev = true
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	app, err := initApp(dev)
	if err != nil {
		log.Fatalf("Failed to init app: %v", err)
	}

	app.registerHandlers()

	log.Printf("Listening on localhost: %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
