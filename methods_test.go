package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	assist_db "assist/db"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"github.com/gorilla/mux"
)

// Test configuration
var (
	testUserId      = "TEST_METHOD_USER"
	testSquadId     = "Super Huge Squad"
	replicantsCount = 1000
	maxThreadsCount = 8
	adb             *assist_db.FirestoreDB
	su              *SessionTestUtil
	ctx             = context.Background()
	router          = mux.NewRouter()
)

// Run flags
var recreate = flag.Bool("recreate", false, "Set this flag to purge the database and create everything from scratch")

// Fake session objects
type SessionTestUtil struct {
}

func (stu *SessionTestUtil) getCurrentUserData(r *http.Request) *assist_db.UserData {
	sd := &assist_db.UserData{
		Admin:    true,
		Status:   assist_db.Admin,
		UserTags: []string{"requester", "approver"},
	}

	return sd
}

func (stu *SessionTestUtil) getCurrentUserID(r *http.Request) string {
	return testUserId
}

func (stu *SessionTestUtil) getCurrentUserRecord(r *http.Request) (*auth.UserRecord, error) {

	return nil, nil
}

// Init app and DB (if recreate flag specified)
func TestInit(t *testing.T) {

	t.Run("Init test app ", func(t *testing.T) {

		assist_db.SetTestPrefix("testMethods_")
		ctx := context.Background()

		// init fireapp
		fireapp, err := firebase.NewApp(ctx, nil)
		if err != nil {
			log.Fatalf("firebase.NewApp: %v", err)
		}

		// init firestore
		adb, err = assist_db.NewFirestoreDB(fireapp, false)
		if err != nil {
			log.Fatalf("Failed to init database: %v", err)
		}

		// init session interfaces
		su := &SessionTestUtil{}

		app := App{
			logWriter: os.Stderr,
			db:        adb,
			sd:        su,
			dev:       false, // set to true if want logs
		}

		app.registerMethodHandlers(router)
	})

	if *recreate {
		t.Run("Clean test DB", func(t *testing.T) {
			err := adb.DeleteCollectionRecurse(ctx, adb.Squads)
			if err != nil {
				t.Fatalf("Failed to clean test data: %v", err)
			}

		})

		t.Run("Create ALL_USERS squad", func(t *testing.T) {
			_, err := adb.Squads.Doc(assist_db.ALL_USERS_SQUAD).Set(ctx, &struct{ Description string }{"Special squad with all users"})
			if err != nil {
				t.Fatalf("Failed to touch ALL_SQUADS doc: %v", err)
			}
		})

		t.Run("Create test user", func(t *testing.T) {
			userInfo := &assist_db.UserInfo{
				DisplayName: "Boris The Blade",
				Email:       "test@mail.com",
				PhoneNumber: "1900555111110",
			}
			err := adb.CreateUser(ctx, testUserId, userInfo, assist_db.Member)
			if err != nil {
				t.Fatalf("Failed to create user: %v", err)
			}
		})

		t.Run("Create his squad", func(t *testing.T) {
			body := strings.NewReader(`{"name": "` + testSquadId + `"}`)
			req, _ := http.NewRequest("POST", "/squads", body)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Result().StatusCode != 200 {
				t.Fatalf("Failed to create squad: %v", rr.Result())
			}
		})

		t.Run("Create replicants", func(t *testing.T) {
			// create channel
			ch := make(chan int)
			var wg sync.WaitGroup

			// launch some working threads
			for i := 0; i < maxThreadsCount; i++ {
				wg.Add(1)

				go func() {
					defer wg.Done()

					for id := range ch {
						body := strings.NewReader(
							fmt.Sprintf(`{
						"displayName": "Rep v.%04d",
						"email": "rep%04d@mail.com",
						"phoneNumber": "+1444555%04d",
						"replicant": true
					}`, id, id, id))
						req, _ := http.NewRequest("POST", "/squads/"+testSquadId+"/members", body)
						rr := httptest.NewRecorder()
						router.ServeHTTP(rr, req)

						if rr.Result().StatusCode != 200 {
							t.Fatalf("Failed to create squad: %v", rr.Result())
						}
					}
				}()
			}

			for i := 0; i < replicantsCount; i++ {
				ch <- i
			}

			close(ch)

			wg.Wait() // wait until all threads will finish
		})
	}
}

// Benchmarking home screen and particular components
func BenchmarkMethodGetHome(b *testing.B) {
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/users/me/home", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Result().StatusCode != 200 {
			b.Fatalf("Failed to retrieve home counters: %v", rr.Result())
		}

	}
}

func BenchmarkGetHome_Home_SquadsCount(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := adb.GetSquadsCount(ctx, testUserId)
		if err != nil {
			b.Fatalf("Failed to retrieve home counters: %v", err)
		}

	}
}

func BenchmarkGetHome_Home_SquadsWithPendingRequests(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := adb.GetSquadsWithPendingRequests(ctx, testUserId, false)
		if err != nil {
			b.Fatalf("Failed to retrieve home counters: %v", err)
		}

	}
}

func BenchmarkGetHome_Home_UserEvents(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := adb.GetUserEvents(ctx, testUserId, 4)
		if err != nil {
			b.Fatalf("Failed to retrieve events: %v", err)
		}

	}
}

func BenchmarkGetHome_Home_AppliedParticipants(b *testing.B) {
	for i := 0; i < b.N; i++ {
		squads, err := adb.GetUserSquads(ctx, testUserId, "admin")
		if err == nil && len(squads) > 0 {
			_, err = adb.GetEventsByStatus(ctx, squads, testUserId, "Applied")
			if err != nil {
				b.Fatalf("Failed to retrieve events: %v", err)
			}
		}
		if err != nil {
			b.Fatalf("Failed to retrieve user squads: %v", err)
		}

	}
}

func BenchmarkGetHome_Home_GetUserRequests(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, err := adb.GetUserRequestQueues(ctx, su.getCurrentUserData(nil).UserTags)
		if err != nil {
			b.Fatalf("Failed to get user requests: %v", err)
		}

	}
}

// Benchmark squad members screen
func BenchmarkMethodGetMembers(b *testing.B) {
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/squads/"+testSquadId+"/members", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Result().StatusCode != 200 {
			b.Fatalf("Failed to retrieve squad members: %v", rr.Result())
		}

	}
}

// Benchmark squad details screen
func BenchmarkMethodGetSquad(b *testing.B) {
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/squads/"+testSquadId, nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Result().StatusCode != 200 {
			b.Fatalf("Failed to retrieve squad info: %v", rr.Result())
		}

	}
}

func BenchmarkGetHome_DBGetSquad(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := adb.GetSquad(ctx, testSquadId)
		if err != nil {
			b.Fatalf("Failed to retrieve squad info: %v", err)
		}

	}
}

func BenchmarkGetHome_DBSquadAdminMembers(b *testing.B) {
	for i := 0; i < b.N; i++ {
		filter := map[string]string{"Status": "Admin"}
		_, err := adb.GetSquadMembers(ctx, testSquadId, nil, &filter)
		if err != nil {
			b.Fatalf("Failed to retrieve squad admins: %v", err)
		}

	}
}
