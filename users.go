package main

import (
	"context"
)

type UserInfo struct {
	DisplayName string
}

type UsersDatabase interface {
	AddUser(ctx context.Context, userId string, userInfo *UserInfo) error
	GetUser(ctx context.Context, userId string) (u *UserInfo, err error)
	AddSquadToMember(ctx context.Context, userId string, squadId string, squadInfo *SquadInfo) error
	DeleteSquadFromMember(ctx context.Context, userId string, squadId string) error
}
