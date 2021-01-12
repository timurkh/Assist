package main

import "context"

type StatusTypes int

const (
	pendingApprove StatusTypes = iota
	guest
	cadet
	member
	instructor
	admin
)

type UserInfo struct {
	DisplayName string
	Email       string
	Phone       string
	Status      StatusTypes
}

type UsersDatabase interface {
	ListUsers(context.Context) ([]*UserInfo, error)
	GetUser(ctx context.Context, id string) (u *UserInfo, err error)
	UpdateUser(ctx context.Context, u *UserInfo) (id string, err error)
}

type Users struct {
	DB UsersDatabase
}
