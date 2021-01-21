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
	*SquadType
}

type SquadsDatabase interface {
	CreateSquad(ctx context.Context, name string, uid string) (squadId string, err error)
	GetSquads(ctx context.Context, userId string) (mySquads []*SquadType, otherSquads []*SquadType, err error)
	DeleteSquad(ctx context.Context, ID string) error
	GetSquad(ctx context.Context, ID string) (*SquadInfo, error)
}
