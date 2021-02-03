package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"assist/db"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"google.golang.org/api/iterator"
)

type App struct {
	authClient *auth.Client
	db         *db.FirestoreDB
}

// GOOGLE_APPLICATION_CREDENTIALS env variable should point to json file with configuration & credentials
func initFirebase(ctx context.Context) *App {

	fireapp, err := firebase.NewApp(ctx, nil)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}

	// Get an auth client from the firebase.App
	authClient, err := fireapp.Auth(ctx)
	if err != nil {
		log.Fatalf("error getting Auth client: %v\n", err)
	}

	dbClient, err := db.NewFirestoreDB(fireapp)
	if err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}

	return &App{authClient, dbClient}
}

func (app *App) listUsers() {

	ctx := context.Background()

	log.Printf("\nUID | DisplayName | Email | EmailVerified | Role\n")
	// Iterating by pages 100 users at a time.
	// Note that using both the Next() function on an iterator and the NextPage()
	// on a Pager wrapping that same iterator will result in an error.
	pager := iterator.NewPager(app.authClient.Users(ctx, ""), 100, "")
	for {
		var users []*auth.ExportedUserRecord
		nextPageToken, err := pager.NextPage(&users)
		if err != nil {
			log.Fatalf("paging error %v\n", err)
		}
		for _, u := range users {
			role := "PendingApprove"
			if u.UserRecord.CustomClaims != nil {
				role = u.UserRecord.CustomClaims["Role"].(string)
			}
			fmt.Printf("%v | %v | %v | %v | %+v\n", u.UserRecord.UID, u.UserRecord.UserInfo.DisplayName, u.UserRecord.UserInfo.Email, u.UserRecord.EmailVerified, role)
		}
		if nextPageToken == "" {
			break
		}
	}
}

func (app *App) updateListOfUsers() {

	ctx := context.Background()

	pager := iterator.NewPager(app.authClient.Users(ctx, ""), 100, "")
	for {
		var users []*auth.ExportedUserRecord
		nextPageToken, err := pager.NextPage(&users)
		if err != nil {
			log.Fatalf("paging error %v\n", err)
		}
		for _, u := range users {
			if u.UserRecord.EmailVerified {
				app.db.UpdateUserInfoFromFirebase(ctx, u.UserRecord)
			}

		}
		if nextPageToken == "" {
			break
		}
	}
}

func (app *App) setRole(uid string, name string) {
	ctx := context.Background()

	// Get current claims
	// Lookup the user associated with the specified uid.
	user, err := app.authClient.GetUser(ctx, uid)
	if err != nil {
		log.Fatal(err)
	}

	if user.CustomClaims == nil {
		user.CustomClaims = make(map[string]interface{})
	}
	user.CustomClaims["Role"] = name

	err = app.authClient.SetCustomUserClaims(ctx, uid, user.CustomClaims)
	if err != nil {
		log.Fatalf("error setting custom claims %v\n", err)
	}

	app.db.UpdateUserStatusFromFirebase(ctx, uid, name)
	if err != nil {
		log.Fatalf("error updating user status %v\n", err)
	}
}

func (app *App) removeCollection(ctx context.Context, collection *firestore.CollectionRef) {
	iter := collection.Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			log.Fatal("Error while iterating through users: %w", err)
		}

		doc.Ref.Delete(ctx)
	}
}

// cycle through users and delete member_squads and own_squads subcollections
func (app *App) flushSquadInfoFromUsers(ctx context.Context) {
	iter := app.db.Users.Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			log.Fatal("Error while iterating through users: %w", err)
		}

		log.Printf("Removing records about squads for user %v", doc.Ref.ID)
		app.removeCollection(ctx, doc.Ref.Collection("member_squads"))
		app.removeCollection(ctx, doc.Ref.Collection("own_squads"))
		app.removeCollection(ctx, doc.Ref.Collection("squads"))
	}
}

// cycle through squads and update information in member_squads and own_squads subcollections of users
func (app *App) populateSquadInfoToUsers(ctx context.Context) {
	iter := app.db.Squads.Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			log.Fatal("Error while iterating through squads: %w", err)
		}

		squadId := doc.Ref.ID
		if squadId == db.ALL_USERS_SQUAD {
			continue
		}

		squadInfo, err := app.db.GetSquad(ctx, squadId)
		if err != nil {
			log.Fatal("Error while getting squads info: %w", err)
		}

		log.Printf("Restoring records about squad %v:", doc.Ref.ID)

		// ensure there is record about squad owner in squad members
		userInfo, err := app.db.GetUser(ctx, squadInfo.Owner)
		if err != nil {
			log.Fatal("Error while obtaining squad owner info: %w", err)
		}
		log.Printf("\towner : %v", squadInfo.Owner)

		squadUserInfo := db.SquadUserInfo{
			UserInfo: *userInfo,
			Status:   db.Owner,
		}
		app.db.AddMemberRecordToSquad(ctx, squadId, squadInfo.Owner, &squadUserInfo)

		if err != nil {
			log.Fatal("Error while populating squad owner info: %w", err)
		}

		// later size will be recalculated while updating user info in squads
		log.Printf("Flushing squad %v size", doc.Ref.ID)
		app.db.FlushSquadSize(ctx, squadId)
		if err != nil {
			log.Fatal("Error while flushing squad %v size: %w", squadId, err)
		}

		squadMembers, err := app.db.GetSquadMembers(ctx, squadId)
		if err != nil {
			log.Fatal("Error while populating squads info to users: %w", err)
		}
		for _, member := range squadMembers {
			memberSquadInfo := &db.MemberSquadInfo{
				SquadInfo: *squadInfo,
				Status:    member.Status,
			}
			err := app.db.AddSquadRecordToMember(ctx, member.ID, squadId, memberSquadInfo)
			if err != nil {
				log.Fatal("Error while populating squads info to users: %w", err)
			}
			log.Printf("\tmember : %v", member.ID)
		}
	}
}

func (app *App) updateUsersInfoInSquads(ctx context.Context) {

	allUsers, err := app.db.GetSquadMembers(ctx, db.ALL_USERS_SQUAD)
	if err != nil {
		log.Fatal("Error while getting list of users: %w", err)
	}

	for _, user := range allUsers {

		log.Printf("Updating user %v details in squads:\n", user.ID)
		userSquads, err := app.db.GetUserSquads(ctx, user.ID)
		if err != nil {
			log.Fatal("Error while getting user %v squads: %w", user.ID, err)
		}
		for squadId, squad := range userSquads {
			log.Printf("\t%v\n", squadId)
			squadUser := &db.SquadUserInfo{
				UserInfo: user.UserInfo,
				Status:   squad.Status,
			}
			app.db.AddMemberRecordToSquad(ctx, squadId, user.ID, squadUser)
		}
	}
}

func (app *App) makeDBConsistent() {
	ctx := context.Background()

	app.updateListOfUsers()
	app.flushSquadInfoFromUsers(ctx)
	app.populateSquadInfoToUsers(ctx)
	app.updateUsersInfoInSquads(ctx)
}

func printUsage() {
	fmt.Printf(
		`USAGE: manage_users <command> [arguments]
	Commands:
	listUsers               - list all users
	setRole <uid> <name>    - expected roles - Member, Admin or empty ("") which will set user pending approve 
	makeDBConsistent		- flush denormalized DB entries stored per user and recreate them from squads collection
`)

}

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		printUsage()
	} else {

		ctx := context.Background()
		app := initFirebase(ctx)

		switch args[0] {
		case "listUsers":
			app.listUsers()
		case "makeDBConsistent":
			app.makeDBConsistent()
		case "setRole":
			app.setRole(args[1], args[2])
		default:
			printUsage()
		}
	}
}
