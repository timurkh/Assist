package main

import (
	"context"
)

type MemberStatus int

const (
	pendingApprove MemberStatus = iota
	member
	admin
)

type UserDetails struct {
	UID    string
	squads map[string]MemberStatus
}

type UsersDatabase interface {
	GetUserDetails(ctx context.Context, id string) (u *UserDetails, err error)
}
