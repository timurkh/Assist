package db

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

func (db *FirestoreDB) CreateSquad(ctx context.Context, squadInfo *SquadInfo) (newSquadId string, err error) {

	newSquadId = db.Squads.NewDoc().ID

	_, err = db.Squads.Doc(newSquadId).Set(ctx, squadInfo)
	if err != nil {
		return newSquadId, fmt.Errorf("Failed to create squad %v: %w", squadInfo.Name, err)
	}

	return newSquadId, nil
}

func (db *FirestoreDB) GetSquads(ctx context.Context, userId string) ([]*SquadInfoRecord, []*SquadInfoRecord, []*SquadInfoRecord, error) {

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

	iter := db.Squads.Documents(ctx)
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

func (db *FirestoreDB) GetUserOwnSquads(ctx context.Context, userID string) (map[string]*SquadInfoRecord, error) {
	return db.GetUserSquads(ctx, userID, "own")
}

func (db *FirestoreDB) GetUserMemberSquads(ctx context.Context, userID string) (map[string]*SquadInfoRecord, error) {
	return db.GetUserSquads(ctx, userID, "member")
}

func (db *FirestoreDB) GetUserSquads(ctx context.Context, userID string, collection string) (map[string]*SquadInfoRecord, error) {

	squads_map := make(map[string]*SquadInfoRecord, 0)

	iter := db.Users.Doc(userID).Collection(collection + "_squads").Documents(ctx)
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

func (db *FirestoreDB) GetSquadMembers(ctx context.Context, squadId string) ([]*SquadUserInfoRecord, error) {

	squadMembers := make([]*SquadUserInfoRecord, 0)

	iter := db.Squads.Doc(squadId).Collection("members").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to get squad members: %w", err)
		}

		s := &SquadUserInfoRecord{}
		err = doc.DataTo(s)
		if err != nil {
			return nil, fmt.Errorf("Failed to get squad members: %w", err)
		}
		s.ID = doc.Ref.ID
		squadMembers = append(squadMembers, s)
	}

	return squadMembers, nil
}

func (db *FirestoreDB) DeleteSquad(ctx context.Context, ID string) error {

	doc, err := db.Squads.Doc(ID).Get(ctx)
	if err != nil {
		return fmt.Errorf("FirestoreDB.DeleteSquad: failed to get squad "+ID+": %w", err)
	}

	owner, err := doc.DataAt("owner")
	if err != nil {
		return fmt.Errorf("Failed to get squad "+ID+" owner: %w", err)
	}

	owner_id := owner.(string)

	//TODO launch go routine to remove this squad from all members
	db.Users.Doc(owner_id).Collection("own_squads").Doc(ID).Delete(ctx)
	if err != nil {
		return fmt.Errorf("Error while deleting records about squad "+ID+" from owner "+owner_id+": %w", err)
	}

	_, err = db.Squads.Doc(ID).Delete(ctx)
	if err != nil {
		return fmt.Errorf("Error while deleting squad "+ID+": %w", err)
	}

	return nil
}

func (db *FirestoreDB) GetSquad(ctx context.Context, ID string) (*SquadInfo, error) {

	doc, err := db.Squads.Doc(ID).Get(ctx)
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

func (db *FirestoreDB) AddMemberToSquad(ctx context.Context, squadId string, userId string, userInfo *SquadUserInfo) error {

	batch := db.Client.Batch()

	docMember := db.Squads.Doc(squadId).Collection("members").Doc(userId)
	batch.Set(docMember, userInfo)
	docSquad := db.Squads.Doc(squadId)
	batch.Update(docSquad, []firestore.Update{
		{Path: "membersCount", Value: firestore.Increment(1)},
	})

	_, err := batch.Commit(ctx)
	if err != nil {
		return fmt.Errorf("Failed to add user "+userId+" to squad "+squadId+": %w", err)
	}

	return nil
}

func (db *FirestoreDB) DeleteMemberFromSquad(ctx context.Context, squadId string, userId string) error {

	batch := db.Client.Batch()

	docMember := db.Squads.Doc(squadId).Collection("members").Doc(userId)
	batch.Delete(docMember)
	docSquad := db.Squads.Doc(squadId)
	batch.Update(docSquad, []firestore.Update{
		{Path: "membersCount", Value: firestore.Increment(-1)},
	})

	_, err := batch.Commit(ctx)
	if err != nil {
		return fmt.Errorf("Failed to delete user "+userId+" from squad "+squadId+": %w", err)
	}

	return nil
}

func (db *FirestoreDB) CheckIfUserIsSquadMember(ctx context.Context, userId string, squadId string) error {

	_, err := db.Squads.Doc(squadId).Collection("members").Doc(userId).Get(ctx)

	return err
}
