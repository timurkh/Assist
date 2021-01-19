package main

import "context"

func (db *firestoreDB) GetUserDetails(ctx context.Context, uid string) (u *UserDetails, err error) {
	return &UserDetails{UID: uid}, nil
}
