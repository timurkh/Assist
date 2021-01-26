package main

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
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

func (db *firestoreDB) GetSquads(ctx context.Context, userId string) ([]*SquadInfoRecord, []*SquadInfoRecord, []*SquadInfoRecord, error) {

	user_own_squads_map, err := db.GetUserOwnSquads(ctx, userId)
	if err != nil {
		return nil, nil, nil, err
	}

	user_member_squads_map, err := db.GetUserMemberSquads(ctx, userId)
	if err != nil {
		return nil, nil, nil, err
	}

	other_squads := make([]*SquadInfoRecord, 0)
	user_own_squads := make([]*SquadInfoRecord, 0, len(user_own_squads_map))
	user_member_squads := make([]*SquadInfoRecord, 0, len(user_member_squads_map))

	iter := db.client.Collection("squads").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, nil, nil, fmt.Errorf("Failed to get squads: %w", err)
		}

		s := &SquadInfoRecord{}
		err = doc.DataTo(s)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("Failed to get squads: %w", err)
		}
		s.ID = doc.Ref.ID

		if _, ok := user_own_squads_map[doc.Ref.ID]; ok {
			user_own_squads = append(user_own_squads, s)
		} else if _, ok := user_member_squads_map[doc.Ref.ID]; ok {
			user_member_squads = append(user_member_squads, s)
		} else {
			other_squads = append(other_squads, s)
		}
	}

	return user_own_squads, user_member_squads, other_squads, nil
}

func (db *firestoreDB) GetUserOwnSquads(ctx context.Context, userID string) (map[string]*SquadInfoRecord, error) {
	return db.GetUserSquads(ctx, userID, "own")
}

func (db *firestoreDB) GetUserMemberSquads(ctx context.Context, userID string) (map[string]*SquadInfoRecord, error) {
	return db.GetUserSquads(ctx, userID, "member")
}

func (db *firestoreDB) GetUserSquads(ctx context.Context, userID string, collection string) (map[string]*SquadInfoRecord, error) {

	squads_map := make(map[string]*SquadInfoRecord, 0)

	iter := db.client.Collection("users").Doc(userID).Collection(collection + "_squads").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to get user squads: %w", err)
		}

		s := &SquadInfoRecord{}
		err = doc.DataTo(s)
		if err != nil {
			return nil, fmt.Errorf("Failed to get user squads: %w", err)
		}
		s.ID = doc.Ref.ID
		squads_map[s.ID] = s
	}

	return squads_map, nil
}

func (db *firestoreDB) GetSquadMembers(ctx context.Context, squadId string) ([]*SquadUserInfo, error) {

	squadMembers := make([]*SquadUserInfo, 0)

	iter := db.client.Collection("squads").Doc(squadId).Collection("members").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to get squad members: %w", err)
		}

		s := &SquadUserInfo{}
		err = doc.DataTo(s)
		if err != nil {
			return nil, fmt.Errorf("Failed to get squad members: %w", err)
		}
		s.ID = doc.Ref.ID
		squadMembers = append(squadMembers, s)
	}

	return squadMembers, nil
}

func (db *firestoreDB) DeleteSquad(ctx context.Context, ID string) error {

	doc, err := db.client.Collection("squads").Doc(ID).Get(ctx)
	if err != nil {
		return fmt.Errorf("FirestoreDB.DeleteSquad: failed to get squad "+ID+": %w", err)
	}

	owner, err := doc.DataAt("owner")
	if err != nil {
		return fmt.Errorf("Failed to get squad "+ID+" owner: %w", err)
	}

	owner_id := owner.(string)

	//TODO launch go routine to remove this squad from all members

	//following part should go to users interface
	db.client.Collection("users").Doc(owner_id).Collection("own_squads").Doc(ID).Delete(ctx)
	if err != nil {
		return fmt.Errorf("Error while deleting records about squad "+ID+" from owner "+owner_id+": %w", err)
	}

	_, err = db.client.Collection("squads").Doc(ID).Delete(ctx)
	if err != nil {
		return fmt.Errorf("Error while deleting squad "+ID+": %w", err)
	}

	return nil
}

func (db *firestoreDB) GetSquad(ctx context.Context, ID string) (*SquadInfo, error) {

	doc, err := db.client.Collection("squads").Doc(ID).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to get squad "+ID+": %w", err)
	}

	s := &SquadInfo{}
	err = doc.DataTo(s)
	if err != nil {
		return nil, fmt.Errorf("Failed to get squad "+ID+": %w", err)
	}

	return s, nil
}

func (db *firestoreDB) AddMemberToSquad(ctx context.Context, squadId string, userId string, userInfo *SquadUserInfo) error {

	batch := db.client.Batch()

	docMember := db.client.Collection("squads").Doc(squadId).Collection("members").Doc(userId)
	batch.Set(docMember, userInfo)
	docSquad := db.client.Collection("squads").Doc(squadId)
	batch.Update(docSquad, []firestore.Update{
		{Path: "membersCount", Value: firestore.Increment(1)},
	})

	_, err := batch.Commit(ctx)
	if err != nil {
		return fmt.Errorf("Failed to add user "+userId+" to squad "+squadId+": %w", err)
	}

	return nil
}

func (db *firestoreDB) DeleteMemberFromSquad(ctx context.Context, squadId string, userId string) error {

	batch := db.client.Batch()

	docMember := db.client.Collection("squads").Doc(squadId).Collection("members").Doc(userId)
	batch.Delete(docMember)
	docSquad := db.client.Collection("squads").Doc(squadId)
	batch.Update(docSquad, []firestore.Update{
		{Path: "membersCount", Value: firestore.Increment(-1)},
	})

	_, err := batch.Commit(ctx)
	if err != nil {
		return fmt.Errorf("Failed to delete user "+userId+" from squad "+squadId+": %w", err)
	}

	return nil
}

func (db *firestoreDB) CheckIfUserIsSquadMember(ctx context.Context, userId string, squadId string) error {

	_, err := db.client.Collection("squads").Doc(squadId).Collection("members").Doc(userId).Get(ctx)

	return err
}
