package main

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
