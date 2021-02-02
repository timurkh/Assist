package db

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
)

const ALL_USERS_SQUAD = "All Users"

type FirestoreDB struct {
	Client *firestore.Client
	Squads *firestore.CollectionRef
	Users  *firestore.CollectionRef
}

// init firestore
func NewFirestoreDB(fireapp *firebase.App) (*FirestoreDB, error) {
	ctx := context.Background()

	dbClient, err := fireapp.Firestore(ctx)
	if err != nil {
		return nil, err
	}

	err = dbClient.RunTransaction(ctx, func(ctx context.Context, t *firestore.Transaction) error {
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("Could not connect to database: %v", err)
	}

	return &FirestoreDB{
		dbClient,
		dbClient.Collection("squads"),
		dbClient.Collection("squads").Doc(ALL_USERS_SQUAD).Collection("members"),
	}, nil
}
