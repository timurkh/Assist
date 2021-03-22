package db

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

const USER_SQUADS = "member_squads"

type MemberStatusType int

const (
	Banned MemberStatusType = iota - 1
	PendingApprove
	Member
	Admin
	Owner
)

var MemberStatusTypes = []MemberStatusType{
	Banned,
	PendingApprove,
	Member,
	Admin,
	Owner,
}

func (s MemberStatusType) String() string {
	switch s {
	case Banned:
		return "Banned"
	case PendingApprove:
		return "Pending Approve"
	case Member:
		return "Member"
	case Admin:
		return "Admin"
	case Owner:
		return "Owner"
	}

	return "Unknown Status"
}

func statusFromString(s string) int {
	for _, t := range MemberStatusTypes {
		if t.String() == s {
			return int(t)
		}
	}
	return -1
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

	if db.dev {
		log.Println("Creating squad " + squadId)
	}

	_, err = db.Squads.Doc(squadId).Create(ctx, map[string]interface{}{
		"Owner":               ownerId,
		"MembersCount":        0,
		"PendingApproveCount": 0,
		"Timestamp":           firestore.ServerTimestamp,
	})
	if err != nil {
		return err
	}

	_, err = db.AddMemberToSquad(ctx, ownerId, squadId, Owner)
	if err != nil {
		return err
	}

	return err
}

func (db *FirestoreDB) GetSquads(ctx context.Context, userId string) ([]string, error) {

	if db.dev {
		log.Println("Getting squads for user " + userId)
	}

	userSquads := make(map[string]bool, 0)
	iterUserSquads := db.Users.Doc(userId).Collection(USER_SQUADS).Select().Documents(ctx)

	defer iterUserSquads.Stop()
	for {
		doc, err := iterUserSquads.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to get user squads: %w", err)
		}

		userSquads[doc.Ref.ID] = true
	}

	otherSquads := make([]string, 0)
	iterOtherSquads := db.Squads.Select().Documents(ctx)
	defer iterOtherSquads.Stop()
	for {
		doc, err := iterOtherSquads.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to get squads: %w", err)
		}

		if doc.Ref.ID == ALL_USERS_SQUAD {
			continue
		}

		if _, ok := userSquads[doc.Ref.ID]; !ok {

			otherSquads = append(otherSquads, doc.Ref.ID)
		}
	}

	return otherSquads, nil
}

func (db *FirestoreDB) GetUserSquads(ctx context.Context, userID string, status string) ([]string, error) {

	squads := make([]string, 0)

	var iter *firestore.DocumentIterator

	if status == "" {
		iter = db.Users.Doc(userID).Collection(USER_SQUADS).Select().Documents(ctx)
	} else if status == "admin" {
		iter = db.Users.Doc(userID).Collection(USER_SQUADS).Where("Status", ">", Admin).Select().Documents(ctx)
	} else {
		return nil, fmt.Errorf("Do not know what to do with status=%v", status)
	}

	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to get user squads: %w", err)
		}

		squads = append(squads, doc.Ref.ID)

	}

	return squads, nil
}

func (db *FirestoreDB) GetUserSquadsMap(ctx context.Context, userID string, status string, includeAllUsersSquad bool) (map[string]*MemberSquadInfoRecord, error) {

	squads_map := make(map[string]*MemberSquadInfoRecord, 0)

	var wg sync.WaitGroup
	var errAllUsers error

	if includeAllUsersSquad {
		var uberSquad MemberSquadInfoRecord
		squads_map[ALL_USERS_SQUAD] = &uberSquad

		wg.Add(1)
		go func() {
			var uberSquadInfo *SquadInfo
			uberSquadInfo, errAllUsers = db.GetSquad(ctx, ALL_USERS_SQUAD)
			uberSquad.ID = ALL_USERS_SQUAD
			uberSquad.SquadInfo = *uberSquadInfo
			uberSquad.Status = Owner
			wg.Done()
		}()
	}

	var iter *firestore.DocumentIterator
	if status == "" {
		iter = db.Users.Doc(userID).Collection(USER_SQUADS).Documents(ctx)
	} else if status == "admin" {
		iter = db.Users.Doc(userID).Collection(USER_SQUADS).Where("Status", ">=", Admin).Documents(ctx)
	} else {
		return nil, fmt.Errorf("Do not know what to do with status=%v", status)
	}

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

	wg.Wait()
	if errAllUsers != nil {
		return nil, fmt.Errorf("Failed to get ALL USERS squad info: %w", errAllUsers)
	}

	return squads_map, nil
}

func (db *FirestoreDB) GetSquadMember(ctx context.Context, squadId string, userId string) (*SquadUserInfo, error) {
	doc, err := db.Squads.Doc(squadId).Collection(MEMBERS).Doc(userId).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to get squad %v member %v: %w", squadId, userId, err)
	}

	s := &SquadUserInfo{}
	err = doc.DataTo(s)
	if err != nil {
		return nil, fmt.Errorf("Failed to get squad %v member %v: %w", squadId, userId, err)
	}

	return s, nil
}

func (db *FirestoreDB) GetSquadMembers(ctx context.Context, squadId string, from *time.Time, filter *map[string]string) ([]*SquadUserInfoRecord, error) {

	if db.dev {
		log.Printf("Getting members of the squad %v\n", squadId)
	}

	squadMembers := make([]*SquadUserInfoRecord, 0)

	iter := db.GetFilteredQuery(db.Squads.Doc(squadId).Collection(MEMBERS), from, filter, statusFromString).Documents(ctx)
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
	return db.DeleteGroup(ctx, "squad", db.Squads, USER_SQUADS, squadId)
}

func (db *FirestoreDB) GetSquad(ctx context.Context, ID string) (*SquadInfo, error) {

	if db.dev {
		log.Println("Getting details for squad " + ID)
	}

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

func (db *FirestoreDB) propagateChangedSquadCounters(squadId string, fields ...string) {
	ctx := context.Background()
	docRef := db.Squads.Doc(squadId)
	doc, err := docRef.Get(ctx)
	if err != nil {
		log.Printf("Failed to get squad %v: %v", squadId, err)
	}

	vals := make([]interface{}, len(fields))
	for i, field := range fields {
		vals[i] = doc.Data()[field]
	}

	iter := docRef.Collection(MEMBERS).Where("Replicant", "!=", true).Documents(ctx)
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
		for i, field := range fields {
			db.updater.dispatchCommand(doc, field, vals[i])
		}
	}
}

func (db *FirestoreDB) getCounter(status MemberStatusType) string {
	path := "MembersCount"
	if status == PendingApprove {
		path = "PendingApproveCount"
	}
	return path
}

func (db *FirestoreDB) addMemberRecordToSquad(ctx context.Context, squadId string, userId string, userInfo *SquadUserInfo) error {

	if db.dev {
		log.Println("Adding member " + userId + " to squad " + squadId)
	}
	batch := db.Client.Batch()

	docSquad := db.Squads.Doc(squadId)
	docMember := docSquad.Collection(MEMBERS).Doc(userId)
	batch.Set(docMember, userInfo)

	batch.Set(docMember, map[string]interface{}{
		"Timestamp": firestore.ServerTimestamp,
	}, firestore.MergeAll)
	batch.Update(docMember, []firestore.Update{
		{Path: "Keys", Value: userInfo.Keys()},
	})
	batch.Update(docSquad, []firestore.Update{
		{Path: db.getCounter(userInfo.Status), Value: firestore.Increment(1)},
	})

	_, err := batch.Commit(ctx)
	if err != nil {
		return fmt.Errorf("Failed to add user "+userId+" to squad "+squadId+": %w", err)
	}

	return nil
}

func (db *FirestoreDB) deleteMemberRecordFromSquad(ctx context.Context, squadId string, userId string) error {

	status, err := db.GetSquadMemberStatus(ctx, userId, squadId)
	if err != nil {
		return err
	}

	batch := db.Client.Batch()
	docSquad := db.Squads.Doc(squadId)
	docMember := docSquad.Collection(MEMBERS).Doc(userId)
	batch.Delete(docMember)
	batch.Update(docSquad, []firestore.Update{
		{Path: db.getCounter(status), Value: firestore.Increment(-1)},
	})

	_, err = batch.Commit(ctx)
	if err != nil {
		return fmt.Errorf("Failed to delete user %v from squad %v: %w", userId, squadId, err)
	}

	go db.propagateChangedSquadCounters(squadId, db.getCounter(status))

	return nil
}

func (db *FirestoreDB) CheckIfUserIsSquadMember(ctx context.Context, userId string, squadId string) error {

	_, err := db.Squads.Doc(squadId).Collection(MEMBERS).Doc(userId).Get(ctx)

	return err
}

func (db *FirestoreDB) FlushSquadSize(ctx context.Context, squadId string) error {

	doc := db.Squads.Doc(squadId)

	snapshotsIter := doc.Collection(MEMBERS).Where("Replicant", "==", true).Snapshots(ctx)
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

	cacheKey := squadId + "/" + userId
	v, found := db.memberStatusCache.Load(cacheKey)
	if found {
		return v.(MemberStatusType), nil
	} else {
		doc, err := db.Squads.Doc(squadId).Collection(MEMBERS).Doc(userId).Get(ctx)
		if err != nil {
			return 0, fmt.Errorf("Failed to get squad "+squadId+": %w", err)
		}
		status, ok := doc.Data()["Status"]
		if ok {
			t := MemberStatusType(status.(int64))
			db.memberStatusCache.Store(cacheKey, t)
			return t, nil
		} else {
			return 0, fmt.Errorf("Failed to get squad " + squadId + " status")
		}
	}
}

func (db *FirestoreDB) SetSquadMemberStatus(ctx context.Context, userId string, squadId string, status MemberStatusType) error {
	oldStatus, err := db.GetSquadMemberStatus(ctx, userId, squadId)
	if err != nil {
		return err
	}

	batch := db.Client.Batch()

	docMemberSquad := db.Users.Doc(userId).Collection(USER_SQUADS).Doc(squadId)
	if squadId != ALL_USERS_SQUAD {
		batch.Update(docMemberSquad, []firestore.Update{{Path: "Status", Value: status}})
	}

	docSquadMember := db.Squads.Doc(squadId).Collection(MEMBERS).Doc(userId)
	batch.Update(docSquadMember, []firestore.Update{{Path: "Status", Value: status}})

	docSquad := db.Squads.Doc(squadId)
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

	if squadId == ALL_USERS_SQUAD {
		db.userDataCache.Delete(userId)
	} else if changedCount != 0 {
		go db.propagateChangedSquadCounters(squadId, "MembersCount", "PendingApproveCount")
	}

	db.memberStatusCache.Delete(squadId + "/" + userId)

	return nil
}

func (db *FirestoreDB) SetSquadMemberNotes(ctx context.Context, userId string, squadId string, notes *map[string]string) error {

	if db.dev {
		log.Printf("Updating note for user '%v' in squad '%v': %+v", userId, squadId, notes)
	}

	docUser := db.Squads.Doc(squadId).Collection(MEMBERS).Doc(userId)

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
	if db.dev {
		log.Println("Creating replicant " + replicantInfo.DisplayName + " in squad " + squadId)
	}

	squadReplicantInfo := &SquadUserInfo{
		UserInfo:  *replicantInfo,
		Replicant: true,
		Status:    Member,
	}

	newReplicantDoc := db.Squads.Doc(squadId).Collection(MEMBERS).NewDoc()

	err = db.addMemberRecordToSquad(ctx, squadId, newReplicantDoc.ID, squadReplicantInfo)
	if err != nil {
		log.Printf("Failed to add replicant record to squad: %v", err)
		return "", err
	}

	if squadId != ALL_USERS_SQUAD {
		go db.propagateChangedSquadCounters(squadId, db.getCounter(Member))
	}

	return newReplicantDoc.ID, nil
}

func (db *FirestoreDB) AddMemberToSquad(ctx context.Context, userId string, squadId string, memberStatus MemberStatusType) (*MemberSquadInfo, error) {
	if db.dev {
		log.Println("Adding user " + userId + " to squad " + squadId)
	}

	userInfo, err := db.GetUser(ctx, userId)
	if err != nil {
		return nil, err
	}

	squadUserInfo := &SquadUserInfo{
		UserInfo: *userInfo,
		Status:   memberStatus,
	}

	err = db.addMemberRecordToSquad(ctx, squadId, userId, squadUserInfo)
	if err != nil {
		return nil, err
	}

	squadInfo, err := db.GetSquad(ctx, squadId)
	if err != nil {
		return nil, err
	}

	memberSquadInfo := &MemberSquadInfo{
		SquadInfo: *squadInfo,
		Status:    memberStatus,
	}

	err = db.addSquadRecordToMember(ctx, userId, squadId, memberSquadInfo)
	if err != nil {
		return nil, err
	}

	if squadId != ALL_USERS_SQUAD {
		go db.propagateChangedSquadCounters(squadId, db.getCounter(memberStatus))
	}

	return memberSquadInfo, nil
}

func (db *FirestoreDB) addSquadRecordToMember(ctx context.Context, userId string, squadId string, squadInfo *MemberSquadInfo) error {

	doc := db.Users.Doc(userId).Collection(USER_SQUADS).Doc(squadId)

	_, err := doc.Set(ctx, squadInfo)
	if err != nil {
		return fmt.Errorf("Failed to add squad to user "+userId+": %w", err)
	}

	return nil
}

func (db *FirestoreDB) DeleteMemberFromSquad(ctx context.Context, userId string, squadId string) error {
	if db.dev {
		log.Println("Removing user " + userId + " from squad " + squadId)
	}

	err := db.deleteMemberRecordFromSquad(ctx, squadId, userId)
	if err != nil {
		return err
	}
	err = db.deleteSquadRecordFromMember(ctx, userId, squadId)
	if err != nil {
		return err
	}

	return nil
}

func (db *FirestoreDB) deleteSquadRecordFromMember(ctx context.Context, userId string, squadId string) error {

	doc := db.Users.Doc(userId).Collection(USER_SQUADS).Doc(squadId)

	_, err := doc.Delete(ctx)
	if err != nil {
		return fmt.Errorf("Failed to delete squad from user "+userId+": %w", err)
	}

	return nil
}
