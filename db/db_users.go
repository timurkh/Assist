package db

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

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

func (db *FirestoreDB) AddSquadToMember(ctx context.Context, userId string, squadId string, squadInfo *MemberSquadInfo) error {

	doc := db.Users.Doc(userId).Collection("squads").Doc(squadId)

	_, err := doc.Set(ctx, squadInfo)
	if err != nil {
		return fmt.Errorf("Failed to add squad to user "+userId+": %w", err)
	}

	return nil
}

func (db *FirestoreDB) AddUser(ctx context.Context, userId string, userInfo *UserInfo) error {

	var sui = SquadUserInfo{
		Status: Member}
	sui.UserInfo = *userInfo
	err := db.AddMemberToSquad(ctx, ALL_USERS_SQUAD, userId, &sui)
	if err != nil {
		return fmt.Errorf("Failed to add user "+userId+": %w", err)
	}

	return nil
}

func (db *FirestoreDB) DeleteSquadFromMember(ctx context.Context, userId string, squadId string) error {

	doc := db.Users.Doc(userId).Collection("squads").Doc(squadId)

	_, err := doc.Delete(ctx)
	if err != nil {
		return fmt.Errorf("Failed to add squad to user "+userId+": %w", err)
	}

	return nil
}

func (db *FirestoreDB) updateUserInfo(ctx context.Context, doc *firestore.DocumentRef, field string, val interface{}) error {
	_, err := doc.Update(ctx, []firestore.Update{
		{
			Path:  field,
			Value: val,
		},
	})
	return err
}

func (db *FirestoreDB) UpdateUser(ctx context.Context, userId string, field string, val interface{}) error {

	docUser := db.Users.Doc(userId)

	err := db.updateUserInfo(ctx, docUser, field, val)
	if err != nil {
		return fmt.Errorf("Failed to update user "+userId+": %w", err)
	}

	go func() {
		ctx := context.Background()
		iter := docUser.Collection("squads").Documents(ctx)
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
			log.Printf("Updating user %v details in squad %v, setting '%v' to '%v':\n", userId, squadId, field, val)
			db.updateUserInfo(ctx, db.Squads.Doc(squadId).Collection("members").Doc(userId), field, val)
			if err != nil {
				log.Printf("Error while updating member %v->%v: %v", squadId, userId, err.Error())
			}
		}
	}()

	return nil
}
