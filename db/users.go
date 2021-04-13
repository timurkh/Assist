package db

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"cloud.google.com/go/firestore"
	"firebase.google.com/go/auth"
	"github.com/patrickmn/go-cache"
	"google.golang.org/api/iterator"
)

type UserInfo struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber"`
}

type UserData struct {
	UID string
	UserInfo
	Status   MemberStatusType
	UserTags []string
	Admin    bool
}

type UserInfoRecord struct {
	ID string `json:"id"`
	UserInfo
}

func getKeys(s string) []string {
	ret := make([]string, 0)
	w := ""

	for _, c := range s {
		w = w + string(c)
		ret = append(ret, w)
	}
	return ret
}

func (ui *UserInfo) Keys() []string {
	ret := make([]string, 0)
	for _, s := range strings.Fields(ui.DisplayName) {
		ret = append(ret, getKeys(strings.ToLower(s))...)
	}
	for _, w := range strings.Split(ui.Email, "@") {
		for _, s := range strings.Fields(w) {
			ret = append(ret, getKeys(strings.ToLower(s))...)
		}
	}
	for _, s := range strings.Fields(ui.PhoneNumber) {
		ret = append(ret, getKeys(s)...)
	}
	return ret
}

func (db *FirestoreDB) GetUserInfo(ctx context.Context, userId string) (u *UserInfo, err error) {
	doc, err := db.Users.Doc(userId).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to get user "+userId+": %w", err)
	}

	s := &UserInfo{}
	doc.DataTo(s)

	return s, nil
}

func (db *FirestoreDB) GetUserData(ctx context.Context, userId string) (sd *UserData, err error) {

	v, found := db.userDataCache.Get(userId)
	if found {
		return v.(*UserData), nil
	} else {

		doc, err := db.Users.Doc(userId).Get(ctx)
		if err != nil {
			return nil, fmt.Errorf("Failed to get user "+userId+": %w", err)
		}

		ud := &UserData{}
		doc.DataTo(ud)
		ud.UID = userId
		ud.Admin = ud.Status == Admin

		db.userDataCache.Set(userId, ud, cache.DefaultExpiration)
		return ud, nil
	}
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

func (db *FirestoreDB) CreateUser(ctx context.Context, userId string, userInfo *UserInfo, status MemberStatusType) error {

	var sui = SquadUserInfo{
		Status: status}
	sui.UserInfo = *userInfo
	err := db.addMemberRecordToSquad(ctx, ALL_USERS_SQUAD, userId, &sui)
	if err != nil {
		return fmt.Errorf("Failed to add user "+userId+": %w", err)
	}

	return nil
}

func (db *FirestoreDB) propagateChangedUserInfo(userId string, field string, val interface{}) {
	ctx := context.Background()
	docUser := db.Users.Doc(userId)
	doc, err := docUser.Get(ctx)
	if err != nil {
		log.Printf("Failed to get user "+userId+": %v", err)
		return
	}

	ui := &UserInfo{}
	doc.DataTo(ui)

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
		db.updater.dispatchCommand(doc, "Keys", ui.Keys())
	}
}

func (db *FirestoreDB) UpdateUser(ctx context.Context, userId string, field string, val interface{}) error {

	docUser := db.Users.Doc(userId)

	err := db.updateDocProperty(ctx, docUser, field, val)
	if err != nil {
		return fmt.Errorf("Failed to update user "+userId+": %w", err)
	}

	go db.propagateChangedUserInfo(userId, field, val)

	db.userDataCache.Delete(userId)

	return nil
}

func (db *FirestoreDB) UpdateUserInfoFromFirebase(ctx context.Context, userRecord *auth.UserRecord) error {
	userId := userRecord.UID
	userData, err := db.GetUserData(ctx, userId)
	if err != nil {
		log.Println("Failed to get user " + userId + " from DB, adding new record to users collection")

		userInfo := &UserInfo{
			DisplayName: userRecord.DisplayName,
			Email:       userRecord.Email,
			PhoneNumber: userRecord.PhoneNumber,
		}

		db.CreateUser(ctx, userId, userInfo, PendingApprove)

		if err != nil {
			return fmt.Errorf("Failed to add user to database: %w", err)
		}
	} else {
		if len(userRecord.Email) > 0 && userData.Email != userRecord.Email {
			db.UpdateUser(ctx, userId, "Email", userRecord.Email)
		}
		if len(userRecord.PhoneNumber) > 0 && userData.PhoneNumber != userRecord.PhoneNumber {
			db.UpdateUser(ctx, userId, "PhoneNumber", userRecord.PhoneNumber)
		}
	}

	db.userDataCache.Delete(userId)

	return nil
}

func (db *FirestoreDB) GetSquadsCount(ctx context.Context, userId string) (interface{}, error) {
	squadsCount := make([]int, len(MemberStatusTypes))
	userSquadsMap, err := db.getUserSquads(ctx, userId)
	if err != nil {
		log.Printf("Error while getting user squads: %v", err)
		return nil, err
	}

	for _, v := range userSquadsMap {
		squadsCount[v]++
	}
	return squadsCount, nil
}

func (db *FirestoreDB) GetSquadsWithPendingRequests(ctx context.Context, userId string, admin bool) (interface{}, error) {
	type squadCount struct {
		Squad string `json:"squad"`
		Count int64  `json:"count"`
	}
	squadsWithRequests := make([]*squadCount, 0)

	var wg sync.WaitGroup
	var errAdminSquad error
	var adminSquadPendingCount int64
	if admin {
		wg.Add(1)
		go func() {
			doc, errAdminSquad := db.Squads.Doc(ALL_USERS_SQUAD).Get(ctx)
			if errAdminSquad == nil {
				pc := doc.Data()["PendingApproveCount"]
				if pc != nil {
					adminSquadPendingCount = pc.(int64)
				}
			}
			wg.Done()
		}()
	}

	iter := db.Users.Doc(userId).Collection(USER_SQUADS).Where("Status", "in", []int{int(Admin), int(Owner)}).Where("PendingApproveCount", "!=", 0).OrderBy("PendingApproveCount", firestore.Desc).Documents(ctx)
	defer iter.Stop()

	for {

		squad, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Failed to get squads with members pending approve: %v", err)
			return nil, err
		}
		sc := &squadCount{squad.Ref.ID, squad.Data()["PendingApproveCount"].(int64)}
		squadsWithRequests = append(squadsWithRequests, sc)
	}

	wg.Wait()

	if errAdminSquad != nil {
		log.Printf("Failed to get counters for ALL_USERS_SQUAD: %v", errAdminSquad)
		return nil, errAdminSquad
	}

	if adminSquadPendingCount > 0 {
		sc := &squadCount{ALL_USERS_SQUAD, adminSquadPendingCount}
		squadsWithRequests = append([]*squadCount{sc}, squadsWithRequests...)
	}

	return squadsWithRequests, nil
}
