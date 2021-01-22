package main

import (
	"context"
	"fmt"
)

func (db *firestoreDB) GetUser(ctx context.Context, userId string) (u *UserInfo, err error) {
	doc, err := db.client.Collection("users").Doc(userId).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to get user "+userId+": %w", err)
	}

	s := &UserInfo{}
	doc.DataTo(s)

	return s, nil
}

func (db *firestoreDB) AddSquadToMember(ctx context.Context, userId string, squadId string, squadInfo *SquadInfo) error {

	doc := db.client.Collection("users").Doc(userId).Collection("member_squads").Doc(squadId)

	_, err := doc.Set(ctx, squadInfo)
	if err != nil {
		return fmt.Errorf("Failed to add squad to user "+userId+": %w", err)
	}

	return nil
}

func (db *firestoreDB) AddUser(ctx context.Context, userId string, userInfo *UserInfo) error {
	doc := db.client.Collection("users").Doc(userId)

	_, err := doc.Set(ctx, userInfo)
	if err != nil {
		return fmt.Errorf("Failed to add user "+userId+": %w", err)
	}

	return nil
}

func (db *firestoreDB) DeleteSquadFromMember(ctx context.Context, userId string, squadId string) error {

	doc := db.client.Collection("users").Doc(userId).Collection("member_squads").Doc(squadId)

	_, err := doc.Delete(ctx)
	if err != nil {
		return fmt.Errorf("Failed to add squad to user "+userId+": %w", err)
	}

	return nil
}
