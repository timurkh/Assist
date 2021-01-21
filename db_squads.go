package main

import (
	"context"
	"fmt"

	"google.golang.org/api/iterator"
)

func (db *firestoreDB) CreateSquad(ctx context.Context, name string, owner string) (newSquadId string, err error) {

	newSquadId = db.client.Collection("squads").NewDoc().ID
	squadDescription := map[string]interface{}{
		"name":         name,
		"owner":        owner,
		"membersCount": 1,
	}

	batch := db.client.Batch()

	batch.Set(db.client.Collection("squads").Doc(newSquadId), squadDescription)
	batch.Set(db.client.Collection("users").Doc(owner).Collection("own_squads").Doc(newSquadId), squadDescription)

	_, err = batch.Commit(ctx)
	if err != nil {
		return "", err
	}

	return newSquadId, nil
}

func (db *firestoreDB) GetSquads(ctx context.Context, userId string) ([]*SquadType, []*SquadType, error) {

	user_squads_map, err := db.GetUserSquads(ctx, userId)
	if err != nil {
		return nil, nil, err
	}

	other_squads := make([]*SquadType, 0)
	user_squads := make([]*SquadType, 0, len(user_squads_map))

	iter := db.client.Collection("squads").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("FirestoreDB: failed to get squads: %v", err)
		}

		s := &SquadType{}
		doc.DataTo(s)
		if err != nil {
			return nil, nil, err
		}
		s.ID = doc.Ref.ID

		if _, ok := user_squads_map[doc.Ref.ID]; !ok {
			other_squads = append(other_squads, s)
		} else {
			user_squads = append(user_squads, s)
		}
	}

	return user_squads, other_squads, nil
}

func (db *firestoreDB) GetUserSquads(ctx context.Context, userID string) (map[string]*SquadType, error) {

	squads_map := make(map[string]*SquadType, 0)

	iter := db.client.Collection("users").Doc(userID).Collection("own_squads").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("FirestoreDB: failed to get user squads: %v", err)
		}

		s := &SquadType{}
		doc.DataTo(s)
		if err != nil {
			return nil, err
		}
		s.ID = doc.Ref.ID
		squads_map[s.ID] = s
	}

	return squads_map, nil
}

func (db *firestoreDB) DeleteSquad(ctx context.Context, ID string) error {

	doc, err := db.client.Collection("squads").Doc(ID).Get(ctx)
	if err != nil {
		return fmt.Errorf("FirestoreDB.DeleteSquad: failed to get squad "+ID+": %v", err)
	}

	owner, err := doc.DataAt("owner")
	if err != nil {
		return fmt.Errorf("FirestoreDB.DeleteSquad: failed to get squad "+ID+" owner: %v", err)
	}

	owner_id := owner.(string)

	db.client.Collection("users").Doc(owner_id).Collection("own_squads").Doc(ID).Delete(ctx)
	if err != nil {
		return fmt.Errorf("FirestoreDB: error while deleting records about squad "+ID+" from owner "+owner_id+": %v", err)
	}

	_, err = db.client.Collection("squads").Doc(ID).Delete(ctx)
	if err != nil {
		return fmt.Errorf("FirestoreDB: error while deleting squad "+ID+": %v", err)
	}

	return nil
}

func (db *firestoreDB) GetSquad(ctx context.Context, ID string) (*SquadInfo, error) {

	doc, err := db.client.Collection("squads").Doc(ID).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("FirestoreDB: failed to get squad "+ID+": %v", err)
	}

	s := &SquadInfo{}
	doc.DataTo(s)

	return s, nil
}
