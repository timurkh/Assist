package db

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/firestore"
	"firebase.google.com/go/auth"
	"google.golang.org/api/iterator"
)

type UserInfo struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber"`
}

type UserInfoRecord struct {
	ID string `json:"id"`
	UserInfo
}

func (db *FirestoreDB) GetUser(ctx context.Context, userId string) (u *UserInfo, err error) {
	doc, err := db.Users.Doc(userId).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to get user "+userId+": %w", err)
	}

	s := &UserInfo{}
	doc.DataTo(s)

	return s, nil
}

func (db *FirestoreDB) GetUserByName(ctx context.Context, userName string) (users []string, err error) {
	users = make([]string, 0)
	iter := db.Users.Where("DisplayName", "==", userName).Documents(ctx)

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			log.Printf("Error while quering users by name: %v", err)
			return users, err
		}

		users = append(users, doc.Ref.ID)
	}

	return users, nil
}

func (db *FirestoreDB) AddSquadRecordToMember(ctx context.Context, userId string, squadId string, squadInfo *MemberSquadInfo) error {

	doc := db.Users.Doc(userId).Collection(USER_SQUADS).Doc(squadId)

	_, err := doc.Set(ctx, squadInfo)
	if err != nil {
		return fmt.Errorf("Failed to add squad to user "+userId+": %w", err)
	}

	return nil
}

func (db *FirestoreDB) CreateUser(ctx context.Context, userId string, userInfo *UserInfo) error {

	var sui = SquadUserInfo{
		Status: Member}
	sui.UserInfo = *userInfo
	err := db.AddMemberRecordToSquad(ctx, ALL_USERS_SQUAD, userId, &sui)
	if err != nil {
		return fmt.Errorf("Failed to add user "+userId+": %w", err)
	}

	return nil
}

func (db *FirestoreDB) DeleteSquadRecordFromMember(ctx context.Context, userId string, squadId string) error {

	doc := db.Users.Doc(userId).Collection(USER_SQUADS).Doc(squadId)

	_, err := doc.Delete(ctx)
	if err != nil {
		return fmt.Errorf("Failed to add squad to user "+userId+": %w", err)
	}

	return nil
}

func (db *FirestoreDB) propagateChangedUserInfo(userId string, field string, val interface{}) {
	ctx := context.Background()
	docUser := db.Users.Doc(userId)
	iter := docUser.Collection(USER_SQUADS).Documents(ctx)
	defer iter.Stop()
	for {
		docSquad, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Error while getting user %v squads: %v", userId, err.Error())
			break
		}

		squadId := docSquad.Ref.ID
		doc := db.Squads.Doc(squadId).Collection("members").Doc(userId)
		db.updater.dispatchCommand(doc, field, val)
	}
}

func (db *FirestoreDB) UpdateUser(ctx context.Context, userId string, field string, val interface{}) error {

	docUser := db.Users.Doc(userId)

	err := db.updateDocProperty(ctx, docUser, field, val)
	if err != nil {
		return fmt.Errorf("Failed to update user "+userId+": %w", err)
	}

	go db.propagateChangedUserInfo(userId, field, val)

	return nil
}

func (db *FirestoreDB) UpdateUserInfoFromFirebase(ctx context.Context, userRecord *auth.UserRecord) error {
	userId := userRecord.UID
	userInfo, err := db.GetUser(ctx, userId)
	if err != nil {
		log.Println("Failed to get user " + userId + " from DB, adding new record to users collection")

		userInfo = &UserInfo{
			DisplayName: userRecord.DisplayName,
			Email:       userRecord.Email,
			PhoneNumber: userRecord.PhoneNumber,
		}
		db.CreateUser(ctx, userId, userInfo)

		if err != nil {
			return fmt.Errorf("Failed to add user to database: %w", err)
		}
	} else {
		if len(userRecord.Email) > 0 && userInfo.Email != userRecord.Email {
			db.UpdateUser(ctx, userId, "Email", userRecord.Email)
		}
		if len(userRecord.PhoneNumber) > 0 && userInfo.PhoneNumber != userRecord.PhoneNumber {
			db.UpdateUser(ctx, userId, "PhoneNumber", userRecord.PhoneNumber)
		}
	}

	var role string
	if r, ok := userRecord.CustomClaims["Role"]; ok {
		role = r.(string)
	}
	db.UpdateUserStatusFromFirebase(ctx, userId, role)

	return nil
}

func (db *FirestoreDB) UpdateUserStatusFromFirebase(ctx context.Context, uid string, role string) error {

	status := PendingApprove
	switch role {
	case "Admin":
		status = Admin
	default:
		status = Member
	}
	_, err := db.Users.Doc(uid).Update(ctx, []firestore.Update{
		{
			Path:  "Status",
			Value: status,
		},
	})
	if err != nil {
		return fmt.Errorf("error while updating user %v status in DB: %v\n", uid, err.Error())
	}

	return nil
}

func (db *FirestoreDB) GetHomeCounters(ctx context.Context, userId string) (map[string]interface{}, error) {

	counters := make(map[string]interface{}, 0)

	//squads
	squads := make([]int, len(MemberStatusTypes))
	for i, status := range MemberStatusTypes {
		iter := db.Users.Doc(userId).Collection(USER_SQUADS).Where("Status", "==", MemberStatusTypes[i]).Snapshots(ctx)
		defer iter.Stop()
		snapshot, err := iter.Next()
		if err != nil {
			log.Printf("Failed to get amount of user %v squads with status %v: %v", userId, status.String(), err)
			return nil, err
		}
		squads[i] = snapshot.Size
	}
	counters["squads"] = squads

	//todo
	{
		iter := db.Users.Doc(userId).Collection(USER_SQUADS).Where("PendingApproveCount", "!=", 0).Snapshots(ctx)
		defer iter.Stop()
		snapshot, err := iter.Next()
		if err != nil {
			log.Printf("Failed to get amount of squads with members pending approve: %v", err)
			return nil, err
		}
		counters["pendingApprove"] = snapshot.Size
	}
	return counters, nil
}
