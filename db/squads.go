package db

type MemberStatusType int

const (
	PendingApproveFromOwner MemberStatusType = iota
	PendingApproveFromMember
	Member
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
	UserInfo
	Status MemberStatusType `json:"status"`
}

type SquadUserInfoRecord struct {
	UserInfoRecord
	Status MemberStatusType `json:"status"`
}
