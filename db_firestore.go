package main

import (
	"cloud.google.com/go/firestore"
)

type firestoreDB struct {
	client *firestore.Client
	squads *firestore.CollectionRef
	users  *firestore.CollectionRef
}

func newFirestoreDB(client *firestore.Client) *firestoreDB {
	return &firestoreDB{
		client,
		client.Collection("squads"),
		client.Collection("squads").Doc(ALL_USERS_SQUAD).Collection("members"),
	}
}
