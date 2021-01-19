package main

import (
	"context"
)

type SquadType struct {
	SquadId      string
	Name         string
	MembersCount int
	MemberStatus MemberStatusType
}

type SquadsDataInterface interface {
	CreateSquad(ctx context.Context, name string, uid string) (squadId string, err error)
	GetSquads(ctx context.Context, userId string) (*[]SquadType, error)
}
