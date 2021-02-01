package db

import (
	"context"
	"fmt"
	"log"

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

func (db *FirestoreDB) GetSquads(ctx context.Context, userId string) ([]*MemberSquadInfoRecord, []*MemberSquadInfoRecord, error) {

	user_squads_map, err := db.GetUserSquads(ctx, userId)
	if err != nil {
		return nil, nil, err
	}

	other_squads := make([]*MemberSquadInfoRecord, 0)
	user_squads := make([]*MemberSquadInfoRecord, 0, len(user_squads_map))

	iter := db.Squads.Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("Failed to get squads: %w", err)
		}

		if memberSI, ok := user_squads_map[doc.Ref.ID]; ok {
			user_squads = append(user_squads, memberSI)
		} else {
			s := &MemberSquadInfo{}
			err = doc.DataTo(s)
			if err != nil {
				return nil, nil, fmt.Errorf("Failed to get squads: %w", err)
			}

			sr := &MemberSquadInfoRecord{
				ID:              doc.Ref.ID,
				MemberSquadInfo: *s,
			}

			other_squads = append(other_squads, sr)
		}
	}

	return user_squads, other_squads, nil
}

func (db *FirestoreDB) GetUserSquads(ctx context.Context, userID string) (map[string]*MemberSquadInfoRecord, error) {

	squads_map := make(map[string]*MemberSquadInfoRecord, 0)

	iter := db.Users.Doc(userID).Collection("squads").Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to get user squads: %w", err)
		}

		s := &MemberSquadInfo{}
		err = doc.DataTo(s)
		if err != nil {
			return nil, fmt.Errorf("Failed to get user squads: %w", err)
		}

		sr := &MemberSquadInfoRecord{
			ID:              doc.Ref.ID,
			MemberSquadInfo: *s,
		}
		squads_map[sr.ID] = sr

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

		s := &SquadUserInfo{}
		err = doc.DataTo(s)
		if err != nil {
			return nil, fmt.Errorf("Failed to get squad members: %w", err)
		}
		sr := &SquadUserInfoRecord{
			ID:            doc.Ref.ID,
			SquadUserInfo: *s,
		}
		squadMembers = append(squadMembers, sr)
	}

	return squadMembers, nil
}

func (db *FirestoreDB) DeleteSquad(ctx context.Context, squadId string) error {

	docSquad := db.Squads.Doc(squadId)

	//delete this squad from all members
	iter := docSquad.Collection("members").Documents(ctx)
	defer iter.Stop()
	for {
		docMember, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error while iterating through squad %v members: %v", squadId, err.Error())
			break
		}

		log.Printf("Deleting squad %v from user %v", squadId, memberId)

		memberId := docMember.Ref.ID
		db.Users.Doc(memberId).Collection("squads").Doc(squadId).Delete(ctx)
	}

	//delete squad itself
	_, err := docSquad.Delete(ctx)
	if err != nil {
		return fmt.Errorf("Error while deleting squad "+squadId+": %w", err)
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
		return fmt.Errorf("Failed to delete user %v from squad %v: %w", userId, squadId, err)
	}

	return nil
}

func (db *FirestoreDB) CheckIfUserIsSquadMember(ctx context.Context, userId string, squadId string) error {

	_, err := db.Squads.Doc(squadId).Collection("members").Doc(userId).Get(ctx)

	return err
}

func (db *FirestoreDB) FlushSquadSize(ctx context.Context, squadId string) error {

	doc := db.Squads.Doc(squadId)

	_, err := doc.Update(ctx, []firestore.Update{
		{
			Path:  "membersCount",
			Value: 0,
		},
	})
	if err != nil {
		return fmt.Errorf("Failed to update squad %v: %w", squadId, err)
	}

	return nil
}
