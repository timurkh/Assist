package main

import (
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

func (db *firestoreDB) getUsersDatabase() UsersDatabase {

	return db
}
