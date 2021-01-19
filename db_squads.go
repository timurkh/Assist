package main

import (
	"context"
	"fmt"

	"google.golang.org/api/iterator"
)

func (db *firestoreDB) CreateSquad(ctx context.Context, name string, owner string) (newSquadId string, err error) {

	newSquadId = db.client.Collection("squads").NewDoc().ID

	batch := db.client.Batch()
	batch.Set(db.client.Collection("squads").Doc(newSquadId), map[string]interface{}{
		"name":         name,
		"owner":        owner,
		"membersCount": 1,
	})

	batch.Set(db.client.Collection("users").Doc(owner).Collection("own_squads").Doc(newSquadId), map[string]interface{}{})

	_, err = batch.Commit(ctx)
	if err != nil {
		return "", err
	}

	return newSquadId, nil
}

func (db *firestoreDB) GetSquads(ctx context.Context, userID string) ([]*SquadType, error) {

	squads := make([]*SquadType, 0)

	iter := db.client.Collection("squads").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("FfirestoreDB: failed to get squads: %v", err)
		}

		s := &SquadType{}
		doc.DataTo(s)
		if err != nil {
			return nil, err
		}
		s.ID = doc.Ref.ID
		squads = append(squads, s)
	}

	return squads, nil
}

func (db *firestoreDB) DeleteSquad(ctx context.Context, ID string) error {

	_, err := db.client.Collection("squads").Doc(ID).Delete(ctx)
	if err != nil {
		return fmt.Errorf("FfirestoreDB: error while deleting squad "+ID+": %v", err)
	}

	return nil
}

func (db *firestoreDB) GetSquad(ctx context.Context, ID string) (*SquadInfo, error) {

	doc, err := db.client.Collection("squads").Doc(ID).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("FirestoreDB: error while deleting squad "+ID+": %v", err)
	}

	s := &SquadInfo{}
	doc.DataTo(s)

	return s, nil
}
