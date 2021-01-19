package main

import (
	"context"
)

type MemberStatusType int

const (
	pendingApprove MemberStatusType = iota
	member
	admin
)

type UserDetails struct {
	UID    string
	squads map[string]MemberStatusType
}

type UsersDatabase interface {
	GetUserDetails(ctx context.Context, id string) (u *UserDetails, err error)
}
