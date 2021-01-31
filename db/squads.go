package db

type MemberStatusType int

const (
	PendingApprove MemberStatusType = iota
	Member
	Admin
	Owner
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
	UserInfo
	Status MemberStatusType `json:"status"`
}

type SquadUserInfoRecord struct {
	ID string `json:"id"`
	SquadUserInfo
}

type MemberSquadInfo struct {
	SquadInfo
	Status MemberStatusType `json:"status"`
}

type MemberSquadInfoRecord struct {
	ID string `json:"id"`
	MemberSquadInfo
}
