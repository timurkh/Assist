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

var testPrefix string = "squads"

func SetTestPrefix(prefix string) {
	testPrefix = prefix
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
		dbClient.Collection(testPrefix + "squads"),
		dbClient.Collection(testPrefix + "squads").Doc(ALL_USERS_SQUAD).Collection("members"),
	}, nil
}

func (db *FirestoreDB) updateDocProperty(ctx context.Context, doc *firestore.DocumentRef, field string, val interface{}) error {
	_, err := doc.Update(ctx, []firestore.Update{
		{
			Path:  field,
			Value: val,
		},
	})
	return err
}
