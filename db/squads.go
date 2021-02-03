package db

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

type MemberStatusType int

const (
	PendingApprove MemberStatusType = iota
	Member
	Admin
	Owner
)

func (s MemberStatusType) String() string {
	texts := []string{
		"Pending Approve",
		"Member",
		"Admin",
		"Owner",
	}

	return texts[s]
}

type SquadInfo struct {
	Owner        string `json:"owner"`
	MembersCount int    `json:"membersCount"`
}

type SquadInfoRecord struct {
	ID string `json:"id"`
	SquadInfo
}

type SquadUserInfo struct {
	UserInfo
	Status MemberStatusType `json:"status"`
}

type SquadUserInfoRecord struct {
	ID string `json:"id"`
	SquadUserInfo
}

type MemberSquadInfo struct {
	SquadInfo
	Status MemberStatusType `json:"status"`
}

type MemberSquadInfoRecord struct {
	ID string `json:"id"`
	MemberSquadInfo
}

func (db *FirestoreDB) CreateSquad(ctx context.Context, squadId string, ownerId string) (err error) {

	squadInfo := &SquadInfo{
		ownerId,
		0,
	}

	_, err = db.Squads.Doc(squadId).Create(ctx, squadInfo)
	if err != nil {
		return err
	}

	err = db.AddMemberToSquad(ctx, ownerId, squadId, Owner)
	if err != nil {
		return err
	}

	return err
}

func (db *FirestoreDB) GetSquads(ctx context.Context, userId string, includeAllUsersSquad bool) ([]*MemberSquadInfoRecord, []*MemberSquadInfoRecord, error) {

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

		if doc.Ref.ID == ALL_USERS_SQUAD {
			continue
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

	if includeAllUsersSquad {
		uberSquadInfo, err := db.GetSquad(ctx, ALL_USERS_SQUAD)
		if err != nil {
			return nil, nil, fmt.Errorf("Failed to get ALL USERS squad info: %w", err)
		}

		var uberSquad MemberSquadInfoRecord
		uberSquad.ID = ALL_USERS_SQUAD
		uberSquad.SquadInfo = *uberSquadInfo
		uberSquad.Status = Owner

		user_squads = append([]*MemberSquadInfoRecord{&uberSquad}, user_squads...)
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

		memberId := docMember.Ref.ID
		log.Printf("Deleting squad %v from user %v", squadId, memberId)

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

func (db *FirestoreDB) propagateChangedSquadInfo(squadId string, field string) {
	ctx := context.Background()
	docSquad, err := db.Squads.Doc(squadId).Get(ctx)
	if err != nil {
		log.Printf("Failed to get squad %v: %v", squadId, err)
	}
	val := docSquad.Data()[field]

	iter := db.Squads.Doc(squadId).Collection("members").Documents(ctx)
	defer iter.Stop()
	for {
		docMember, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error while getting squad %v member: %v", squadId, err.Error())
			break
		}

		userId := docMember.Ref.ID
		log.Printf("Updating squad %v details for member %v, setting '%v' to '%v':\n", squadId, userId, field, val)

		db.updateDocProperty(ctx, db.Users.Doc(userId).Collection("squads").Doc(squadId), field, val)
		if err != nil {
			log.Printf("Error while updating squad %v->%v: %v", userId, squadId, err.Error())
		}
	}
}

func (db *FirestoreDB) AddMemberRecordToSquad(ctx context.Context, squadId string, userId string, userInfo *SquadUserInfo) error {

	batch := db.Client.Batch()

	docMember := db.Squads.Doc(squadId).Collection("members").Doc(userId)
	batch.Set(docMember, userInfo)
	docSquad := db.Squads.Doc(squadId)
	batch.Update(docSquad, []firestore.Update{
		{Path: "MembersCount", Value: firestore.Increment(1)},
	})

	_, err := batch.Commit(ctx)
	if err != nil {
		return fmt.Errorf("Failed to add user "+userId+" to squad "+squadId+": %w", err)
	}

	go db.propagateChangedSquadInfo(squadId, "MembersCount")

	return nil
}

func (db *FirestoreDB) DeleteMemberRecordFromSquad(ctx context.Context, squadId string, userId string) error {

	batch := db.Client.Batch()

	docMember := db.Squads.Doc(squadId).Collection("members").Doc(userId)
	batch.Delete(docMember)
	docSquad := db.Squads.Doc(squadId)
	batch.Update(docSquad, []firestore.Update{
		{Path: "MembersCount", Value: firestore.Increment(-1)},
	})

	_, err := batch.Commit(ctx)
	if err != nil {
		return fmt.Errorf("Failed to delete user %v from squad %v: %w", userId, squadId, err)
	}

	go db.propagateChangedSquadInfo(squadId, "MembersCount")

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
			Path:  "MembersCount",
			Value: 0,
		},
	})
	if err != nil {
		return fmt.Errorf("Failed to update squad %v: %w", squadId, err)
	}

	return nil
}

func (db *FirestoreDB) GetSquadMemberStatus(ctx context.Context, userId string, squadId string) (MemberStatusType, error) {
	doc, err := db.Squads.Doc(squadId).Collection("members").Doc(userId).Get(ctx)
	if err != nil {
		return 0, fmt.Errorf("Failed to get squad "+squadId+": %w", err)
	}
	status, ok := doc.Data()["Status"]
	if ok {
		return MemberStatusType(status.(int64)), nil
	} else {
		return 0, fmt.Errorf("Failed to get squad " + squadId + " status")
	}
}

func (db *FirestoreDB) SetSquadMemberStatus(ctx context.Context, userId string, squadId string, status MemberStatusType) error {
	docSquad := db.Users.Doc(userId).Collection("squads").Doc(squadId)
	err := db.updateDocProperty(ctx, docSquad, "Status", status)
	if err != nil {
		return fmt.Errorf("Failed to update user "+userId+" status: %w", err)
	}

	docMember := db.Squads.Doc(squadId).Collection("members").Doc(userId)
	err = db.updateDocProperty(ctx, docMember, "Status", status)
	if err != nil {
		return fmt.Errorf("Failed to update user "+userId+" status: %w", err)
	}
	return nil
}

func (db *FirestoreDB) AddMemberToSquad(ctx context.Context, userId string, squadId string, memberStatus MemberStatusType) error {
	log.Println("Adding user " + userId + " to squad " + squadId)

	userInfo, err := db.GetUser(ctx, userId)
	if err != nil {
		return err
	}

	squadUserInfo := &SquadUserInfo{
		UserInfo: *userInfo,
		Status:   memberStatus,
	}

	err = db.AddMemberRecordToSquad(ctx, squadId, userId, squadUserInfo)
	if err != nil {
		return err
	}

	squadInfo, err := db.GetSquad(ctx, squadId)
	if err != nil {
		return err
	}

	memberSquadInfo := &MemberSquadInfo{
		SquadInfo: *squadInfo,
		Status:    memberStatus,
	}

	err = db.AddSquadRecordToMember(ctx, userId, squadId, memberSquadInfo)
	if err != nil {
		return err
	}

	return nil
}

func (db *FirestoreDB) DeleteMemberFromSquad(ctx context.Context, userId string, squadId string) error {
	log.Println("Removing user " + userId + " from squad " + squadId)

	err := db.DeleteSquadRecordFromMember(ctx, userId, squadId)
	if err != nil {
		return err
	}

	err = db.DeleteMemberRecordFromSquad(ctx, squadId, userId)
	if err != nil {
		return err
	}
	return nil
}
