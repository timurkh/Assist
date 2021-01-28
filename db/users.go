package db

type UserInfo struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber"`
}

type UserInfoRecord struct {
	ID string `json:"id"`
	UserInfo
}
