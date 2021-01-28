package db

import (
	"context"
	"fmt"
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

func (db *FirestoreDB) AddSquadToUser(ctx context.Context, userId string, collection string, squadId string, squadInfo *SquadInfo) error {

	doc := db.Users.Doc(userId).Collection(collection + "_squads").Doc(squadId)

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

	doc := db.Users.Doc(userId).Collection("member_squads").Doc(squadId)

	_, err := doc.Delete(ctx)
	if err != nil {
		return fmt.Errorf("Failed to add squad to user "+userId+": %w", err)
	}

	return nil
}
