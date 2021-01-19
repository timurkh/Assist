package main

import (
	"context"

	"cloud.google.com/go/firestore"
)

type firestoreDB struct {
	client *firestore.Client
}

func newFirestoreDB(client *firestore.Client) *firestoreDB {
	return &firestoreDB{
		client,
	}
}

// Ensure firestoreDB implements UsersDatabase
var _ UsersDatabase = &firestoreDB{}

func (db *firestoreDB) getUsersDatabase() UsersDatabase {

	return db
}

func (db *firestoreDB) GetUserDetails(ctx context.Context, uid string) (u *UserDetails, err error) {
	return &UserDetails{UID: uid}, nil
}

//
func (db *firestoreDB) AddSquad(ctx context.Context, name string, owner string) (squadId string, err error) {

	doc, _, err := db.client.Collection("squads").Add(ctx, map[string]interface{}{
		"name":         name,
		"membersCount": 1,
	})

	return doc.ID, err
}
