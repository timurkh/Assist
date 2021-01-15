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
