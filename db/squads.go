package db

type MemberStatusType int

const (
	PendingApprove MemberStatusType = iota
	Member
	Admin
	Owner
)

func (s MemberStatusType) String() string {
	texts := []string{
		"Pending Approve",
		"Member",
		"Admin",
		"Owner",
	}

	return texts[s]
}

type SquadInfo struct {
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
