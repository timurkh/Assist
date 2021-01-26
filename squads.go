package main

import (
	"context"
)

type MemberStatusType int

const (
	pendingApproveFromOwner MemberStatusType = iota
	pendingApproveFromMember
	member
	//admin - will add some time later
)

type SquadInfo struct {
	Name         string `json:"name"`
	Owner        string `json:"owner"`
	MembersCount int    `json:"membersCount"`
}

type SquadInfoRecord struct {
	ID string `json:"id"`
	SquadInfo
}

type SquadUserInfo struct {
	UserInfoRecord
	Status MemberStatusType `json:"status"`
}

type SquadsDatabase interface {
	GetSquad(ctx context.Context, ID string) (*SquadInfo, error)
	GetSquads(ctx context.Context, userId string) (ownSquads []*SquadInfoRecord, memberSquads []*SquadInfoRecord, otherSquads []*SquadInfoRecord, err error)
	GetSquadMembers(ctx context.Context, squadId string) ([]*SquadUserInfo, error)
	CreateSquad(ctx context.Context, name string, uid string) (squadId string, err error)
	DeleteSquad(ctx context.Context, ID string) error
	AddMemberToSquad(ctx context.Context, squadId string, userId string, userInfo *SquadUserInfo) error
	DeleteMemberFromSquad(ctx context.Context, squadId string, userId string) error
	CheckIfUserIsSquadMember(ctx context.Context, userId string, squadId string) error
}
