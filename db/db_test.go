package db

import (
	"context"
	"testing"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
)

func TestDB(t *testing.T) {

	ctx := context.Background()

	// init fireapp
	fireapp, err := firebase.NewApp(ctx, nil)
	if err != nil {
		t.Errorf("firebase.NewApp: %v", err)
	}

	// init firestore
	dbClient, err := fireapp.Firestore(ctx)
	if err != nil {
		t.Errorf("fireapp.Firestore: %v", err)
	}

	err = dbClient.RunTransaction(ctx, func(ctx context.Context, t *firestore.Transaction) error {
		return nil
	})

	if err != nil {
		t.Errorf("firestoredb: could not connect: %v", err)
	}

	newFirestoreDB(dbClient)
}
