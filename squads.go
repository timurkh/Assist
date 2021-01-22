package main

import (
	"context"
)

type SquadType struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Owner        string `json:"owner"`
	MembersCount int    `json:"membersCount"`
}

type SquadInfo struct {
	SquadType
}

type SquadsDatabase interface {
	GetSquad(ctx context.Context, ID string) (*SquadInfo, error)
	GetSquads(ctx context.Context, userId string) (ownSquads []*SquadType, memberSquads []*SquadType, otherSquads []*SquadType, err error)
	CreateSquad(ctx context.Context, name string, uid string) (squadId string, err error)
	DeleteSquad(ctx context.Context, ID string) error
	AddMemberToSquad(ctx context.Context, squadId string, userId string, userInfo *UserInfo) error
	DeleteMemberFromSquad(ctx context.Context, squadId string, userId string) error
}
