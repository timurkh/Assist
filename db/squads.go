package db

import (
	"context"
	"fmt"
	"log"
	"time"

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

var MemberStatusTypes = []MemberStatusType{
	PendingApprove,
	Member,
	Admin,
	Owner,
}

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
	Owner               string `json:"owner"`
	MembersCount        int    `json:"membersCount"`
	PendingApproveCount int    `json:"pendingApproveCount"`
}

type SquadInfoRecord struct {
	ID string `json:"id"`
	SquadInfo
}

type SquadUserInfo struct {
	UserInfo
	Replicant bool              `json:"replicant"`
	Status    MemberStatusType  `json:"status"`
	Tags      []string          `json:"tags"`
	Notes     map[string]string `json:"notes"`
	Timestamp interface{}       `json:"timestamp"`
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

	log.Println("Creating squad " + squadId)

	_, err = db.Squads.Doc(squadId).Create(ctx, map[string]interface{}{
		"Owner":               ownerId,
		"MembersCount":        0,
		"PendingApproveCount": 0,
		"Timestamp":           firestore.ServerTimestamp,
	})
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

	log.Println("Getting squads for user " + userId)

	user_squads_map, err := db.GetUserSquads(ctx, userId)
	if err != nil {
		return nil, nil, err
	}

	other_squads := make([]*MemberSquadInfoRecord, 0)
	user_squads := make([]*MemberSquadInfoRecord, 0, len(user_squads_map))

	iter := db.Squads.OrderBy("Timestamp", firestore.Asc).Documents(ctx)
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

	iter := db.Users.Doc(userID).Collection(USER_SQUADS).Documents(ctx)
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

func (db *FirestoreDB) GetSquadMembers(ctx context.Context, squadId string, from string) ([]*SquadUserInfoRecord, error) {

	numRecords := 10
	log.Printf("Getting members of the squad %v\n", squadId)

	squadMembers := make([]*SquadUserInfoRecord, 0)

	query := db.Squads.Doc(squadId).Collection("members").OrderBy("Timestamp", firestore.Asc)
	if from != "" {
		log.Printf("\tstarting from %v\n", from)
		timeFrom, err := time.Parse(time.RFC3339, from)
		if err != nil {
			return nil, fmt.Errorf("Failed to convert from to a time struct: %w", err)
		}

		query = query.StartAfter(timeFrom)
	}

	iter := query.Limit(numRecords).Documents(ctx)
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
		log.Printf("%+v\n", sr)
	}

	return squadMembers, nil
}

func (db *FirestoreDB) DeleteSquad(ctx context.Context, squadId string) error {

	log.Println("Deleting squad " + squadId)

	docSquad := db.Squads.Doc(squadId)

	//delete this squad from all members
	go func() {
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

			db.Users.Doc(memberId).Collection(USER_SQUADS).Doc(squadId).Delete(ctx)
		}
	}()

	//delete squad itself
	err := db.deleteDocRecurse(ctx, docSquad)
	if err != nil {
		return fmt.Errorf("Error while deleting squad "+squadId+": %w", err)
	}

	return nil
}

func (db *FirestoreDB) GetSquad(ctx context.Context, ID string) (*SquadInfo, error) {

	log.Println("Getting details for squad " + ID)

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

	iter := db.Squads.Doc(squadId).Collection("members").Where("Replicant", "!=", true).Documents(ctx)
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

		doc := db.Users.Doc(userId).Collection(USER_SQUADS).Doc(squadId)
		db.updater.dispatchCommand(doc, field, val)
	}
}

func (db *FirestoreDB) AddMemberRecordToSquad(ctx context.Context, squadId string, userId string, userInfo *SquadUserInfo) error {

	batch := db.Client.Batch()

	docMember := db.Squads.Doc(squadId).Collection("members").Doc(userId)
	batch.Set(docMember, userInfo)

	docSquad := db.Squads.Doc(squadId)
	path := "MembersCount"
	if userInfo.Status == PendingApprove {
		path = "PendingApproveCount"
	}

	batch.Set(docMember, map[string]interface{}{
		"Timestamp": firestore.ServerTimestamp,
	}, firestore.MergeAll)
	batch.Update(docSquad, []firestore.Update{
		{Path: path, Value: firestore.Increment(1)},
	})

	_, err := batch.Commit(ctx)
	if err != nil {
		return fmt.Errorf("Failed to add user "+userId+" to squad "+squadId+": %w", err)
	}

	if squadId != ALL_USERS_SQUAD {
		go db.propagateChangedSquadInfo(squadId, path)
	}

	return nil
}

func (db *FirestoreDB) DeleteMemberRecordFromSquad(ctx context.Context, squadId string, userId string) error {

	status, err := db.GetSquadMemberStatus(ctx, userId, squadId)
	if err != nil {
		return err
	}
	path := "MembersCount"
	if status == PendingApprove {
		path = "PendingApproveCount"
	}

	batch := db.Client.Batch()
	docSquad := db.Squads.Doc(squadId)
	docMember := docSquad.Collection("members").Doc(userId)
	batch.Delete(docMember)
	batch.Update(docSquad, []firestore.Update{
		{Path: path, Value: firestore.Increment(-1)},
	})

	_, err = batch.Commit(ctx)
	if err != nil {
		return fmt.Errorf("Failed to delete user %v from squad %v: %w", userId, squadId, err)
	}

	go db.propagateChangedSquadInfo(squadId, path)

	return nil
}

func (db *FirestoreDB) CheckIfUserIsSquadMember(ctx context.Context, userId string, squadId string) error {

	_, err := db.Squads.Doc(squadId).Collection("members").Doc(userId).Get(ctx)

	return err
}

func (db *FirestoreDB) FlushSquadSize(ctx context.Context, squadId string) error {

	doc := db.Squads.Doc(squadId)

	snapshotsIter := doc.Collection("members").Where("Replicant", "==", true).Snapshots(ctx)
	defer snapshotsIter.Stop()
	snapshot, err := snapshotsIter.Next()

	if err != nil {
		log.Fatalf("Failed to get amount of replicants in squad %v: %v", squadId, err)
	}

	replicantsAmount := snapshot.Size

	_, err = doc.Update(ctx, []firestore.Update{
		{
			Path:  "MembersCount",
			Value: replicantsAmount,
		},
		{
			Path:  "PendingApproveCount",
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
	oldStatus, err := db.GetSquadMemberStatus(ctx, userId, squadId)
	if err != nil {
		return err
	}

	batch := db.Client.Batch()

	docSquad := db.Users.Doc(userId).Collection(USER_SQUADS).Doc(squadId)
	batch.Update(docSquad, []firestore.Update{{Path: "Status", Value: status}})

	docMember := db.Squads.Doc(squadId).Collection("members").Doc(userId)
	batch.Update(docMember, []firestore.Update{{Path: "Status", Value: status}})
	err = db.updateDocProperty(ctx, docMember, "Status", status)

	changedCount := 0
	if oldStatus == PendingApprove && status != PendingApprove {
		changedCount = 1
	}
	if oldStatus != PendingApprove && status == PendingApprove {
		changedCount = -1
	}
	if changedCount != 0 {
		batch.Update(docSquad, []firestore.Update{
			{Path: "MembersCount", Value: firestore.Increment(changedCount)},
		})
		batch.Update(docSquad, []firestore.Update{
			{Path: "PendingApproveCount", Value: firestore.Increment(-changedCount)},
		})
	}

	_, err = batch.Commit(ctx)
	if err != nil {
		return fmt.Errorf("Failed to change user "+userId+" status: %w", err)
	}

	return nil
}

func (db *FirestoreDB) SetSquadMemberNotes(ctx context.Context, userId string, squadId string, notes *map[string]string) error {

	log.Printf("Updating note for user '%v' in squad '%v': %+v", userId, squadId, notes)

	docUser := db.Squads.Doc(squadId).Collection("members").Doc(userId)

	_, err := docUser.Update(ctx, []firestore.Update{
		{Path: "Notes", Value: notes},
	})

	if err != nil {
		log.Printf("Failed to update user notes: %v", err)
		return err
	}

	return nil
}

func (db *FirestoreDB) CreateReplicant(ctx context.Context, replicantInfo *UserInfo, squadId string) (replicantId string, err error) {
	log.Println("Creating replicant " + replicantInfo.DisplayName + " in squad " + squadId)

	squadReplicantInfo := &SquadUserInfo{
		UserInfo:  *replicantInfo,
		Replicant: true,
		Status:    Member,
	}
	docSquad := db.Squads.Doc(squadId)
	members := docSquad.Collection("members")
	newReplicantDoc := members.NewDoc()

	batch := db.Client.Batch()
	batch.Set(newReplicantDoc, squadReplicantInfo)
	batch.Set(newReplicantDoc, map[string]interface{}{
		"Timestamp": firestore.ServerTimestamp,
	}, firestore.MergeAll)
	batch.Update(docSquad, []firestore.Update{
		{Path: "MembersCount", Value: firestore.Increment(1)},
	})

	_, err = batch.Commit(ctx)
	if err != nil {
		return "", fmt.Errorf("Failed to add replicant '%v' to squad '%v': %w", replicantInfo, squadId, err)
	}

	// use same counter for real users and replicants for now
	go db.propagateChangedSquadInfo(squadId, "MembersCount")

	return newReplicantDoc.ID, nil
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
