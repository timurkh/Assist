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

func (db *firestoreDB) ListUsers(context.Context) ([]*UserInfo, error) {

	return []*UserInfo{&UserInfo{
		"TestName",
		"as@sa.ru",
		"7-555-1234567",
		pendingApprove,
	}}, nil
}

func (db *firestoreDB) GetUser(ctx context.Context, id string) (u *UserInfo, err error) {
	return &UserInfo{
		"TestName",
		"as@sa.ru",
		"7-555-1234567",
		pendingApprove,
	}, nil
}

func (db *firestoreDB) UpdateUser(ctx context.Context, u *UserInfo) (id string, err error) {
	return "test_id", nil
}
