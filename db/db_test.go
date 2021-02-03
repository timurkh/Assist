package db

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
)

func TestInitDB(t *testing.T) {

	ctx := context.Background()

	var db *FirestoreDB

	t.Run("Init test firestore DB", func(t *testing.T) {
		fireapp, err := firebase.NewApp(ctx, nil)
		if err != nil {
			t.Fatalf("firebase.NewApp: %v", err)
		}

		dbClient, err := fireapp.Firestore(ctx)
		if err != nil {
			t.Fatalf("fireapp.Firestore: %v", err)
		}

		err = dbClient.RunTransaction(ctx, func(ctx context.Context, t *firestore.Transaction) error {
			return nil
		})

		if err != nil {
			t.Fatalf("firestoredb: could not connect: %v", err)
		}

		SetTestPrefix("test_")

		db, err = NewFirestoreDB(fireapp)
		if err != nil {
			t.Fatalf("NewFirestoreDB failed: %v", err)
		}
	})

	t.Run("Clean test DB", func(t *testing.T) {
		err := db.deleteCollectionRecurse(ctx, db.Squads)
		if err != nil {
			t.Fatalf("Failed to clean test data: %v", err)
		}

	})

	t.Run("Create ALL_USERS squad", func(t *testing.T) {
		_, err := db.Squads.Doc(ALL_USERS_SQUAD).Set(ctx, &struct{ Description string }{"Special squad with all users"})
		if err != nil {
			t.Fatalf("Failed to touch ALL_SQUADS doc: %v", err)
		}
	})

	t.Run("Create test users", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			userInfo := &UserInfo{
				DisplayName: fmt.Sprint("User ", i),
				Email:       fmt.Sprint(i, "test@mail.com"),
				PhoneNumber: fmt.Sprint("1900555111110", i),
			}
			err := db.CreateUser(ctx, fmt.Sprint("TEST_USER_", i), userInfo)
			if err != nil {
				t.Fatalf("Failed to create user: %v", err)
			}
		}
	})

	t.Run("Create test superuser", func(t *testing.T) {
		userInfo := &UserInfo{
			DisplayName: "SuperUser",
			Email:       "test@mail.com",
			PhoneNumber: "19005550000000",
		}
		err := db.CreateUser(ctx, "SUPER_USER", userInfo)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
	})

	t.Run("Create test squads owned by superuser", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			err := db.CreateSquad(ctx, fmt.Sprint("TEST_SQUAD_", i), "SUPER_USER")
			if err != nil {
				t.Fatalf("Failed to create squad: %v", err)
			}
		}
	})

	t.Run("Add test users to one squad", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			err := db.AddMemberToSquad(ctx, fmt.Sprint("TEST_USER_", i), "TEST_SQUAD_0", Member)
			if err != nil {
				t.Fatalf("Failed to add user to squad: %v", err)
			}
		}
	})

	t.Run("Check GetSquadMembers", func(t *testing.T) {
		allUsers, err := db.GetSquadMembers(ctx, ALL_USERS_SQUAD)
		if err != nil {
			t.Errorf("Failed to retrieve squad members: %v", err)
		}

		if len(allUsers) != 6 {
			t.Errorf("Wrong number of members in ALL USERS squad")
		}

		testSquad0, err := db.GetSquadMembers(ctx, "TEST_SQUAD_0")
		if len(testSquad0) != 6 {
			t.Errorf("Wrong number of members in TEST_SQUAD_0 squad - %v", len(testSquad0))
		}

		testSquad1, err := db.GetSquadMembers(ctx, "TEST_SQUAD_1")
		if len(testSquad1) != 1 {
			t.Errorf("Wrong number of members in TEST_SQUAD_1 squad - %v", len(testSquad1))
		}
	})

	t.Run("Check GetSquads", func(t *testing.T) {
		own_squads, other_squads, err := db.GetSquads(ctx, "SUPER_USER", true)
		if err != nil {
			t.Errorf("Failed to retrieve squads: %v", err)
		}

		if len(own_squads) != 3 {
			t.Errorf("Wrong number of own_squads for SUPER_USER - %v", len(own_squads))
		}

		if len(other_squads) != 0 {
			t.Errorf("Wrong number of other_squads for SUPER_USER - %v", len(other_squads))
		}

		for i := 15; i >= 0; i-- {
			own_squads, other_squads, err = db.GetSquads(ctx, "TEST_USER_0", false)
			if err != nil {
				t.Errorf("Failed to retrieve squads: %v", err)
			}
			if len(own_squads) != 1 {
				t.Errorf("Wrong number of own_squads for TEST_USER_0 - %v", len(own_squads))
			}
			if len(other_squads) != 1 {
				t.Errorf("Wrong number of other_squads for TEST_USER_0 - %v", len(other_squads))
			}

			if own_squads[0].MembersCount != 6 || own_squads[0].ID != "TEST_SQUAD_0" || own_squads[0].Owner != "SUPER_USER" || own_squads[0].Status != Member {
				if i != 0 {
					t.Logf("Wrong squad info for the only squad for TEST_USER_0 - %+v", own_squads[0])
					time.Sleep(100 * time.Millisecond)
					t.Log("Trying once more")
				} else {
					t.Errorf("Wrong squad info for the only squad for TEST_USER_0 - %+v", own_squads[0])
				}
			} else {
				break
			}

		}
	})

	t.Run("Check DeleteMemberFromSquad", func(t *testing.T) {
		//delete user from squad
		err := db.DeleteMemberFromSquad(ctx, "TEST_USER_0", "TEST_SQUAD_0")
		if err != nil {
			t.Errorf("Failed to delete member from squad: %v", err)
		}

		// ensure squad has correct records about users
		testSquad0, err := db.GetSquadMembers(ctx, "TEST_SQUAD_0")
		if len(testSquad0) != 5 {
			t.Errorf("Wrong number of members in TEST_SQUAD_0 squad after deleting TEST_USER_0 from it - %v", len(testSquad0))
		}

		// ensure user has correct records about squads
		own_squads, other_squads, err := db.GetSquads(ctx, "TEST_USER_0", false)
		if err != nil {
			t.Errorf("Failed to retrieve squads: %v", err)
		}
		if len(own_squads) != 0 {
			t.Errorf("Wrong number of own_squads for TEST_USER_0 - %v", len(own_squads))
		}
		if len(other_squads) != 2 {
			t.Errorf("Wrong number of other_squads for TEST_USER_0 - %v", len(other_squads))
		}

		// ensure another user has correct records about squads
		for i := 15; i >= 0; i-- {
			own_squads, other_squads, err = db.GetSquads(ctx, "TEST_USER_1", false)
			if err != nil {
				t.Errorf("Failed to retrieve squads: %v", err)
			}
			if len(own_squads) != 1 {
				t.Errorf("Wrong number of own_squads for TEST_USER_1 - %v", len(own_squads))
			}
			if len(other_squads) != 1 {
				t.Errorf("Wrong number of other_squads for TEST_USER_1 - %v", len(own_squads))
			}

			if own_squads[0].MembersCount != 5 || own_squads[0].ID != "TEST_SQUAD_0" || own_squads[0].Owner != "SUPER_USER" || own_squads[0].Status != Member {
				if i != 0 {
					t.Logf("Wrong squad info for the only squad for TEST_USER_1 - %+v", own_squads[0])
					time.Sleep(100 * time.Millisecond)
					t.Log("Trying once more")
				} else {
					t.Errorf("Wrong squad info for the only squad for TEST_USER_1 - %+v", own_squads[0])
				}
			} else {
				break
			}
		}
	})

	t.Run("Check DeleteSquad", func(t *testing.T) {
		//delete user from squad
		err := db.DeleteSquad(ctx, "TEST_SQUAD_0")
		if err != nil {
			t.Errorf("Failed to delete member from squad: %v", err)
		}

		own_squads, other_squads, err := db.GetSquads(ctx, "SUPER_USER", true)
		if err != nil {
			t.Errorf("Failed to retrieve squads: %v", err)
		}

		if len(own_squads) != 2 {
			t.Errorf("Wrong number of own_squads for SUPER_USER - %v", len(own_squads))
		}

		if len(other_squads) != 0 {
			t.Errorf("Wrong number of other_squads for SUPER_USER - %v", len(other_squads))
		}

		own_squads, other_squads, err = db.GetSquads(ctx, "TEST_USER_1", false)
		if err != nil {
			t.Errorf("Failed to retrieve squads: %v", err)
		}
		if len(own_squads) != 0 {
			t.Errorf("Wrong number of own_squads for TEST_USER_1 - %v", len(own_squads))
		}
		if len(other_squads) != 1 {
			t.Errorf("Wrong number of other_squads for TEST_USER_1 - %v", len(other_squads))
		}
	})
}
