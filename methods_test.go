package main

import (
	assist_db "assist/db"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	firebase "firebase.google.com/go"
	"github.com/gorilla/mux"
)

var testUserId = "TEST_METHOD_USER"
var testSquadId = "Super Huge Squad"
var db *assist_db.FirestoreDB
var ctx = context.Background()
var router = mux.NewRouter()

type SessionTestUtil struct {
}

func (stu *SessionTestUtil) getSessionData(r *http.Request) *SessionData {
	sd := &SessionData{
		Admin: true,
	}

	return sd
}

func (stu *SessionTestUtil) getCurrentUserID(r *http.Request) string {
	return testUserId
}

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
		db, err = assist_db.NewFirestoreDB(fireapp)
		if err != nil {
			log.Fatalf("Failed to init database: %v", err)
		}

		// init session interfaces
		su := &SessionTestUtil{}

		app := App{
			os.Stderr,
			db,
			su,
			nil, //SessionMiddleware is not required
			true,
		}

		app.registerMethodHandlers(router)
	})

	t.Run("Clean test DB", func(t *testing.T) {
		err := db.DeleteCollectionRecurse(ctx, db.Squads)
		if err != nil {
			t.Fatalf("Failed to clean test data: %v", err)
		}

	})

	t.Run("Create ALL_USERS squad", func(t *testing.T) {
		_, err := db.Squads.Doc(assist_db.ALL_USERS_SQUAD).Set(ctx, &struct{ Description string }{"Special squad with all users"})
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
		err := db.CreateUser(ctx, testUserId, userInfo)
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
}

func TestCreateReplicants(t *testing.T) {
	for i := 0; i < 100; i++ {
		body := strings.NewReader(
			fmt.Sprintf(`{
				"displayName": "Rep v.%04d",
				"email": "rep%04d@mail.com",
				"phoneNumber": "+1444555%04d",
				"replicant": true
			}`, i, i, i))
		req, _ := http.NewRequest("POST", "/squads/"+testSquadId+"/members", body)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Result().StatusCode != 200 {
			t.Fatalf("Failed to create squad: %v", rr.Result())
		}
	}
}
