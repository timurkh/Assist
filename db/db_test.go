package db

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
)

var db *FirestoreDB
var ctx = context.Background()

func TestInitDB(t *testing.T) {

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

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

		db, err = NewFirestoreDB(fireapp, false)
		if err != nil {
			t.Fatalf("NewFirestoreDB failed: %v", err)
		}
	})
}

func TestCleanDB(t *testing.T) {
	t.Run("Clean test DB", func(t *testing.T) {
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			err := db.DeleteCollectionRecurse(ctx, db.Squads)
			if err != nil {
				t.Fatalf("Failed to clean test data: %v", err)
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			err := db.DeleteCollectionRecurse(ctx, db.Events)
			if err != nil {
				t.Fatalf("Failed to clean test data: %v", err)
			}
		}()
		wg.Wait()
	})

	t.Run("Create ALL_USERS squad", func(t *testing.T) {
		_, err := db.Squads.Doc(ALL_USERS_SQUAD).Set(ctx, &struct{ Description string }{"Special squad with all users"})
		if err != nil {
			t.Fatalf("Failed to touch ALL_SQUADS doc: %v", err)
		}
	})
}

func TestSquads(t *testing.T) {
	t.Run("Create test users", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			userInfo := &UserInfo{
				DisplayName: fmt.Sprint("User ", i),
				Email:       fmt.Sprint(i, "test@mail.com"),
				PhoneNumber: fmt.Sprint("1900555111110", i),
			}
			err := db.CreateUser(ctx, fmt.Sprint("TEST_USER_", i), userInfo, Member)
			if err != nil {
				t.Fatalf("Failed to create user: %v", err)
			}
		}
	})

	t.Run("Create users", func(t *testing.T) {
		var wg sync.WaitGroup

		for i := 0; i < 2; i++ {
			wg.Add(1)

			go func(i int) {
				defer wg.Done()
				userInfo := &UserInfo{
					DisplayName: fmt.Sprint("Pending Approve User ", i),
					Email:       fmt.Sprint(i, "pending@mail.com"),
					PhoneNumber: fmt.Sprint("1900555111120", i),
				}
				err := db.CreateUser(ctx, fmt.Sprint("PENDING_APPROVE_USER_", i), userInfo, Member)
				if err != nil {
					t.Fatalf("Failed to create pending approve user: %v", err)
				}
			}(i)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			userInfo := &UserInfo{
				DisplayName: "SuperUser",
				Email:       "test@mail.com",
				PhoneNumber: "19005550000000",
			}
			err := db.CreateUser(ctx, "SUPER_USER", userInfo, Member)
			if err != nil {
				t.Fatalf("Failed to create user: %v", err)
			}
		}()

		wg.Wait()
	})

	t.Run("Create test squads owned by superuser", func(t *testing.T) {
		var wg sync.WaitGroup

		for i := 0; i < 2; i++ {
			wg.Add(1)

			go func(i int) {
				defer wg.Done()
				err := db.CreateSquad(ctx, fmt.Sprint("TEST_SQUAD_", i), "SUPER_USER")
				if err != nil {
					t.Fatalf("Failed to create squad: %v", err)
				}
			}(i)
		}
		wg.Wait()
	})

	t.Run("Add test users to one squad", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(1)

			go func(i int) {
				defer wg.Done()
				squad, err := db.AddMemberToSquad(ctx, fmt.Sprint("TEST_USER_", i), "TEST_SQUAD_0", Member)
				if err != nil {
					t.Fatalf("Failed to add user to squad: %v", err)
				}
				if squad.Status != Member {
					t.Fatalf("AddMemberToSquad returned wrong squad info, expected status=Member, memberCount=%v and recieved: %+v", i+2, squad)
				}
			}(i)
		}

		// add couple more with status pending approve
		for i := 0; i < 2; i++ {
			wg.Add(1)

			go func(i int) {
				defer wg.Done()
				_, err := db.AddMemberToSquad(ctx, fmt.Sprint("PENDING_APPROVE_USER_", i), "TEST_SQUAD_0", PendingApprove)
				if err != nil {
					t.Fatalf("Failed to add user to squad: %v", err)
				}
			}(i)
		}
		wg.Wait()
	})

	t.Run("Check GetSquadMembers", func(t *testing.T) {
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			allUsers, err := db.GetSquadMembers(ctx, ALL_USERS_SQUAD, nil, nil)
			if err != nil {
				t.Errorf("Failed to retrieve squad members: %v", err)
			}

			if len(allUsers) != 8 {
				t.Errorf("Wrong number of members in ALL USERS squad")
			}
			wg.Done()
		}()

		wg.Add(1)
		go func() {
			testSquad0, err := db.GetSquadMembers(ctx, "TEST_SQUAD_0", nil, nil)
			if err != nil {
				t.Errorf("Failed to retrieve squad members: %v", err)
			}
			if len(testSquad0) != 8 {
				t.Errorf("Wrong number of members in TEST_SQUAD_0 squad - %v", len(testSquad0))
			}
			wg.Done()
		}()

		wg.Add(1)
		go func() {
			testSquad1, err := db.GetSquadMembers(ctx, "TEST_SQUAD_1", nil, nil)
			if err != nil {
				t.Errorf("Failed to retrieve squad members: %v", err)
			}
			if len(testSquad1) != 1 {
				t.Errorf("Wrong number of members in TEST_SQUAD_1 squad - %v", len(testSquad1))
			}
			wg.Done()
		}()

		wg.Wait()
	})

	t.Run("Add replicants users to another squad", func(t *testing.T) {
		var wg sync.WaitGroup

		for i := 0; i < 5; i++ {
			wg.Add(1)

			go func(i int) {
				defer wg.Done()
				userInfo := &UserInfo{
					DisplayName: fmt.Sprintf("Replicant #%v", i),
					Email:       fmt.Sprintf("id%v@replicants.org", i),
					PhoneNumber: "19005550000000",
				}
				_, err := db.CreateReplicant(ctx, userInfo, "TEST_SQUAD_1")
				if err != nil {
					t.Fatalf("Failed to create replicant: %v", err)
				}
			}(i)
		}
		wg.Wait()
		testSquad1, err := db.GetSquadMembers(ctx, "TEST_SQUAD_1", nil, nil)
		if err != nil {
			t.Fatalf("Failed to get squad members: %v", err)
		}

		if len(testSquad1) != 6 {
			t.Errorf("Wrong number of members in TEST_SQUAD_1 squad - %v", len(testSquad1))
		}

		if testSquad1[2].Status != Member {
			t.Errorf("Wrong status for newly created replicant - %v, should be %v", testSquad1[2].Status, Member)
		}
	})

	t.Run("Check GetSquads", func(t *testing.T) {
		testSquad := "TEST_SQUAD_0"

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			ownSquads, err := db.GetUserSquadsMap(ctx, "SUPER_USER", "", true)
			if err != nil {
				t.Errorf("Failed to retrieve squads: %v", err)
			}

			if len(ownSquads) != 3 {
				t.Errorf("Wrong number of ownSquads for SUPER_USER - %v", len(ownSquads))
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			otherSquads, err := db.GetOtherSquads(ctx, "SUPER_USER")

			if err != nil {
				t.Errorf("Failed to retrieve squads: %v", err)
			}
			if len(otherSquads) != 0 {
				t.Errorf("Wrong number of otherSquads for SUPER_USER - %v", len(otherSquads))
			}

		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			otherSquads, err := db.GetOtherSquads(ctx, "TEST_USER_0")
			if err != nil {
				t.Errorf("Failed to retrieve squads: %v", err)
			}
			if len(otherSquads) != 1 {
				t.Errorf("Wrong number of otherSquads for TEST_USER_0 - %v", len(otherSquads))
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 5; i >= 0; i-- {
				ownSquads, err := db.GetUserSquadsMap(ctx, "TEST_USER_0", "", false)
				if err != nil {
					t.Errorf("Failed to retrieve squads: %v", err)
				}
				if len(ownSquads) != 1 {
					t.Errorf("Wrong number of ownSquads for TEST_USER_0 - %v", len(ownSquads))
				}

				if ownSquads[testSquad].MembersCount != 6 || ownSquads[testSquad].PendingApproveCount != 2 || ownSquads[testSquad].Owner != "SUPER_USER" || ownSquads[testSquad].Status != Member {
					if i != 0 {
						t.Logf("Wrong squad info for the only squad for TEST_USER_0 - %+v", ownSquads[testSquad])
						time.Sleep(100 * time.Millisecond)
						t.Log("Trying once more")
					} else {
						t.Errorf("Wrong squad info for the only squad for TEST_USER_0 - %+v", ownSquads[testSquad])
					}
				} else {
					break
				}
			}
		}()
		wg.Wait()
	})

	t.Run("Check DeleteMemberFromSquad", func(t *testing.T) {
		testSquad := "TEST_SQUAD_0"

		var wg sync.WaitGroup

		//delete user from squad
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := db.DeleteMemberFromSquad(ctx, "TEST_USER_0", testSquad)
			if err != nil {
				t.Errorf("Failed to delete member from squad: %v", err)
			}
		}()

		//delete pending approve from squad
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := db.DeleteMemberFromSquad(ctx, "PENDING_APPROVE_USER_0", testSquad)
			if err != nil {
				t.Errorf("Failed to delete member from squad: %v", err)
			}
		}()
		wg.Wait()

		// ensure squad has correct records about users
		wg.Add(1)
		go func() {
			defer wg.Done()
			testSquad0, err := db.GetSquadMembers(ctx, testSquad, nil, nil)
			if err != nil {
				t.Errorf("Failed to retrieve squads: %v", err)
			}
			if len(testSquad0) != 6 {
				t.Errorf("Wrong number of members in %v squad after deleting TEST_USER_0 from it - %v", testSquad, len(testSquad0))
			}
		}()

		// ensure user has correct records about squads
		wg.Add(1)
		go func() {
			defer wg.Done()
			ownSquads, err := db.GetUserSquads(ctx, "TEST_USER_0", "")
			if err != nil {
				t.Errorf("Failed to retrieve squads: %v", err)
			}
			if len(ownSquads) != 0 {
				t.Errorf("Wrong number of ownSquads for TEST_USER_0 - %v", len(ownSquads))
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			otherSquads, err := db.GetOtherSquads(ctx, "TEST_USER_0")
			if err != nil {
				t.Errorf("Failed to retrieve squads: %v", err)
			}
			if len(otherSquads) != 2 {
				t.Errorf("Wrong number of otherSquads for TEST_USER_0 - %v", len(otherSquads))
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			otherSquads, err := db.GetOtherSquads(ctx, "TEST_USER_1")
			if err != nil {
				t.Errorf("Failed to retrieve squads: %v", err)
			}
			if len(otherSquads) != 1 {
				t.Errorf("Wrong number of otherSquads for TEST_USER_1 - %v", len(otherSquads))
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			// ensure another user has correct records about squads
			for i := 5; i >= 0; i-- {
				ownSquads, err := db.GetUserSquadsMap(ctx, "TEST_USER_1", "", false)
				if err != nil {
					t.Errorf("Failed to retrieve squads: %v", err)
				}
				if len(ownSquads) != 1 {
					t.Errorf("Wrong number of ownSquads for TEST_USER_1 - %v", len(ownSquads))
				}

				if ownSquads[testSquad].MembersCount != 5 || ownSquads[testSquad].PendingApproveCount != 1 || ownSquads[testSquad].Owner != "SUPER_USER" || ownSquads[testSquad].Status != Member {
					if i != 0 {
						t.Logf("Wrong squad info for the only squad for TEST_USER_1 - %+v", ownSquads[testSquad])
						time.Sleep(100 * time.Millisecond)
						t.Log("Trying once more")
					} else {
						t.Errorf("Wrong squad info for the only squad for TEST_USER_1 - %+v", ownSquads[testSquad])
					}
				} else {
					break
				}
			}
		}()
		wg.Wait()
	})

	t.Run("Check SetSquadMemberStatus", func(t *testing.T) {
		testUser := "PENDING_APPROVE_USER_1"
		testSquad := "TEST_SQUAD_0"

		//delete user from squad
		err := db.SetSquadMemberStatus(ctx, testUser, testSquad, Member)
		if err != nil {
			t.Errorf("Failed to set status member: %v", err)
		}

		// ensure squad members have correct records about squads
		for i := 5; i >= 0; i-- {
			ownSquads, err := db.GetUserSquadsMap(ctx, testUser, "", false)
			if err != nil {
				t.Errorf("Failed to retrieve squads: %v", err)
			}

			if ownSquads[testSquad].MembersCount != 6 || ownSquads[testSquad].PendingApproveCount != 0 || ownSquads[testSquad].Owner != "SUPER_USER" || ownSquads[testSquad].Status != Member {
				if i != 0 {
					t.Logf("Wrong squad info for the only squad for %v - %+v", testUser, ownSquads[testSquad])
					time.Sleep(100 * time.Millisecond)
					t.Log("Trying once more")
				} else {
					t.Errorf("Wrong squad info for the only squad for %v - %+v", testUser, ownSquads[testSquad])
				}
			} else {
				break
			}
		}

		// ensure records in squad are also correct
		squadInfo, err := db.GetSquad(ctx, testSquad)
		if err != nil {
			t.Errorf("Failed to retrieve squadInfo: %v", err)
		}
		if squadInfo.MembersCount != 6 || squadInfo.PendingApproveCount != 0 {
			t.Errorf("Wrong squad info for squad %v - %+v", testSquad, squadInfo)
		}

		// set back user status to PendingApprove
		err = db.SetSquadMemberStatus(ctx, "PENDING_APPROVE_USER_1", "TEST_SQUAD_0", PendingApprove)
		if err != nil {
			t.Errorf("Failed to set status member: %v", err)
		}

		// ensure squad members have correct records about squads
		for i := 5; i >= 0; i-- {
			ownSquads, err := db.GetUserSquadsMap(ctx, "TEST_USER_1", "", false)
			if err != nil {
				t.Errorf("Failed to retrieve squads: %v", err)
			}

			if ownSquads[testSquad].MembersCount != 5 || ownSquads[testSquad].PendingApproveCount != 1 || ownSquads[testSquad].Owner != "SUPER_USER" || ownSquads[testSquad].Status != Member {
				if i != 0 {
					t.Logf("Wrong squad info for the only squad for TEST_USER_1 - %+v", ownSquads[testSquad])
					time.Sleep(100 * time.Millisecond)
					t.Log("Trying once more")
				} else {
					t.Errorf("Wrong squad info for the only squad for TEST_USER_1 - %+v", ownSquads[testSquad])
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

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			ownSquadsMap, err := db.GetUserSquadsMap(ctx, "SUPER_USER", "", true)
			if err != nil {
				t.Errorf("Failed to retrieve squads: %v", err)
			}

			if len(ownSquadsMap) != 2 {
				t.Errorf("Wrong number of ownSquads for SUPER_USER - %v", len(ownSquadsMap))
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			otherSquads, err := db.GetOtherSquads(ctx, "SUPER_USER")
			if err != nil {
				t.Errorf("Failed to retrieve squads: %v", err)
			}
			if len(otherSquads) != 0 {
				t.Errorf("Wrong number of otherSquads for SUPER_USER - %v", len(otherSquads))
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			ownSquads, err := db.GetUserSquads(ctx, "TEST_USER_1", "")
			if err != nil {
				t.Errorf("Failed to retrieve squads: %v", err)
			}
			if len(ownSquads) != 0 {
				t.Errorf("Wrong number of ownSquads for TEST_USER_1 - %v", len(ownSquads))
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			otherSquads, err := db.GetOtherSquads(ctx, "TEST_USER_1")
			if err != nil {
				t.Errorf("Failed to retrieve squads: %v", err)
			}
			if len(otherSquads) != 1 {
				t.Errorf("Wrong number of otherSquads for TEST_USER_1 - %v", len(otherSquads))
			}
		}()
		wg.Wait()
	})

	t.Run("Add test users to one squad", func(t *testing.T) {
		var wg sync.WaitGroup

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(i int) {
				squad, err := db.AddMemberToSquad(ctx, fmt.Sprint("TEST_USER_", i), "TEST_SQUAD_1", Member)
				if err != nil {
					t.Fatalf("Failed to add user to squad: %v", err)
				}
				if squad.Status != Member {
					t.Errorf("AddMemberToSquad returned wrong squad info: %+v", squad)
				}
				wg.Done()
			}(i)
		}
		wg.Wait()
	})

	t.Run("Tag users", func(t *testing.T) {
		tag := Tag{
			Name:   "tag",
			Values: map[string]int64{"v1": 0, "v2": 0, "v3": 0, "v4": 0, "v5": 0},
		}
		err := db.CreateTag(ctx, "TEST_SQUAD_1", &tag)
		if err != nil {
			t.Fatalf("Failed to create tag: %v", err)
		}

		var wg sync.WaitGroup

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(i int) {
				_, err := db.SetSquadMemberTag(ctx, fmt.Sprint("TEST_USER_", i), "TEST_SQUAD_1", "tag", fmt.Sprint("v", i))
				if err != nil {
					t.Fatalf("Failed to tag user: %v", err)
				}
				wg.Done()
			}(i)
		}

		wg.Wait()
	})
}

func TestEvents(t *testing.T) {
	var eventIds [5]string
	t.Run("Create events", func(t *testing.T) {
		squadId := "TEST_SQUAD_1"
		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			tm := time.Date(2025, time.Month(3), 8+i, 11, 00, 0, 0, time.UTC)
			if i == 3 {
				squadId = "NON_EXISTANT_SQUAD"
			}
			wg.Add(1)
			go func(i int, squadId string, tm *time.Time) {
				eventInfo := &EventInfo{
					Date:     tm,
					TimeFrom: "10:00",
					TimeTo:   "15:00",
					Text:     fmt.Sprint("Super event ", i),
					SquadId:  squadId,
					OwnerId:  fmt.Sprint("TEST_USER_", i),
				}
				var err error
				eventIds[i], err = db.CreateEvent(ctx, eventInfo)
				if err != nil {
					t.Fatalf("Failed to create event: %v", err)
				}
				wg.Done()
			}(i, squadId, &tm)
		}
		wg.Wait()

	})

	var eventInfo *EventInfo
	t.Run("Register for one of the events", func(t *testing.T) {
		var err error
		eventInfo, err = db.GetEvent(ctx, eventIds[1])
		if err != nil {
			t.Fatalf("Failed to get event info : %v", err)
		}

		if eventInfo.Text != "Super event 1" {
			t.Fatalf("Wrong event info")
		}
		userIds := []string{"TEST_USER_0", "TEST_USER_2"}
		err = db.RegisterParticipants(ctx, userIds, eventIds[1], eventInfo, Applied)
		if err != nil {
			t.Fatalf("Failed to register users %v for event : %v", strings.Join(userIds, "&"), err)
		}
	})

	t.Run("Getting list of events", func(t *testing.T) {
		squadIds := []string{"TEST_SQUAD_1"}
		events, err := db.GetEvents(ctx, squadIds, "TEST_USER_0")
		if err != nil {
			t.Fatalf("Failed to get events: %v", err)
		}
		if len(events) != 3 {
			t.Fatalf("Wrong number of events, expected 3 but recieved %v", len(events))
		}

		if events[1].Status != Applied {
			t.Fatalf("Wrong status for user %v in event %v, expected Applied, recieved %v", "TEST_SQUAD_0", events[1].Text, events[1].Status)
		}

		if events[1].Applied != 2 || events[1].Going != 0 {
			t.Fatalf("Wrong numbers for applied & going to event %v, expected 2, 0 but recieved %v, %v", events[1].Text, events[1].Applied, events[1].Going)
		}
	})

	t.Run("Getting event participants", func(t *testing.T) {
		filter := map[string]string{"Tag": "tag/v2"}
		participants, err := db.GetParticipants(ctx, eventIds[1], nil, &filter)
		if err != nil {
			t.Fatalf("Failed to get participants: %v", err)
		}
		if len(participants) != 1 {
			t.Fatalf("Wrong number of participants, expected 1 but recieved %v", len(participants))
		}
		if participants[0].ID != "TEST_USER_2" {
			t.Fatalf("Wrong participant, expected TEST_USER_2 but recieved %v", participants[0].ID)
		}
	})

	var candidateIds []string
	var wg sync.WaitGroup
	t.Run("Get event candidates, register some of them", func(t *testing.T) {

		wg.Add(1)
		go func() {
			filter := map[string]string{"Keys": "Re"}
			candidates, err := db.GetCandidates(ctx, "TEST_SQUAD_1", eventIds[1], "", &filter)
			if err != nil {
				t.Fatalf("Failed to get event candidates: %v", err)
			}
			if len(candidates) != 5 {
				t.Fatalf("Wrong number of candidates, expected 5 (all replicants), recieved %v", len(candidates))
			}

			candidateIds = []string{candidates[0].ID, candidates[1].ID}
			err = db.RegisterParticipants(ctx, candidateIds, eventIds[1], eventInfo, Going)

			wg.Done()
		}()

		wg.Add(1)
		go func() {
			filter := map[string]string{"Keys": "Us"}
			candidates, err := db.GetCandidates(ctx, "TEST_SQUAD_1", eventIds[1], "", &filter)
			if err != nil {
				t.Fatalf("Failed to get event candidates: %v", err)
			}
			if len(candidates) != 3 {
				t.Fatalf("Wrong number of candidates, expected 3 (SUPER_USER, TEST_USER_3, TEST_USER_4), recieved %v", len(candidates))
			}
			wg.Done()
		}()
	})

	t.Run("Change participant status", func(t *testing.T) {
		wg.Add(1)
		go func() {
			err := db.SetParticipantStatus(ctx, "TEST_USER_0", eventIds[1], Going)
			if err != nil {
				t.Fatalf("Failed to change participant status: %v", err)
			}
			wg.Done()
		}()
	})
	wg.Wait()

	t.Run("Check event counters", func(t *testing.T) {
		squadIds := []string{"TEST_SQUAD_1"}
		events, err := db.GetEvents(ctx, squadIds, "TEST_USER_1")
		if err != nil {
			t.Fatalf("Failed to get event info : %v", err)
		}
		if events[1].Applied != 1 || events[1].Going != 3 {
			t.Fatalf("Wrong numbers for applied & going to event %v, expected 1, 3 but recieved %v, %v", events[1].Text, events[1].Applied, events[1].Going)
		}
	})

	t.Run("Check delete participants", func(t *testing.T) {
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			err := db.DeleteParticipant(ctx, "TEST_USER_0", eventIds[1])
			if err != nil {
				t.Fatalf("Failed to remove TEST_USER_0 from event : %v", err)
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			err := db.DeleteParticipant(ctx, candidateIds[0], eventIds[1])
			if err != nil {
				t.Fatalf("Failed to remove replicant from event : %v", err)
			}
		}()
		wg.Wait()

		wg.Add(1)
		go func() {
			defer wg.Done()
			filter := map[string]string{"Tag": "tag/v0"}
			candidates, err := db.GetCandidates(ctx, "TEST_SQUAD_1", eventIds[1], "", &filter)
			if err != nil {
				t.Fatalf("Failed to get event candidates: %v", err)
			}
			if len(candidates) != 1 {
				t.Fatalf("Wrong number of candidates, expected 1 (TEST_USER_0), recieved %v", len(candidates))
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			squadIds := []string{"TEST_SQUAD_1"}
			events, err := db.GetEvents(ctx, squadIds, "TEST_USER_1")
			if err != nil {
				t.Fatalf("Failed to get event info : %v", err)
			}
			if events[1].Applied != 1 || events[1].Going != 1 {
				t.Fatalf("Wrong numbers for applied & going to event %v, expected 1, 1 but recieved %v, %v", events[1].Text, events[1].Applied, events[1].Going)
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			events, err := db.GetUserEvents(ctx, "TEST_USER_2", 3)
			if err != nil {
				t.Fatalf("Failed to get event info : %v", err)
			}

			if len(events) != 2 {
				t.Fatalf("Wrong numbers for events for TEST_USER_2, expected 1, but recieved %v", len(events))
			}
		}()
		wg.Wait()
	})

}
