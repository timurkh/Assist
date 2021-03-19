package main

import (
	assist_db "assist/db"
	"log"
	"net/http"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

var (
	loginTmpl        = parseBodyTemplate("login.html")
	homeTmpl         = parseBodyTemplate("home.html")
	userinfoTmpl     = parseBodyTemplate("userinfo.html")
	squadsTmpl       = parseBodyTemplate("squads.html")
	squadMembersTmpl = parseBodyTemplate("squadMembers.html")
	squadDetailsTmpl = parseBodyTemplate("squadDetails.html")
	squadNotesTmpl   = parseBodyTemplate("squadNotes.html")
	eventsTmpl       = parseBodyTemplate("events.html")
	participantsTmpl = parseBodyTemplate("eventParticipants.html")
	aboutTmpl        = parseAboutTemplate()
)

func (app *App) squadsHandler(w http.ResponseWriter, r *http.Request) error {

	return squadsTmpl.ExecuteWithSession(app, w, r, Values{})
}

func (app *App) squadDetailsHandler(w http.ResponseWriter, r *http.Request) error {
	params := mux.Vars(r)
	squadId := params["squadId"]

	_, level := app.checkAuthorization(r, "me", squadId, squadMember|squadAdmin|squadOwner|systemAdmin)

	if level >= squadAdmin {
		return squadDetailsTmpl.ExecuteWithSession(app, w, r, Values{
			"SquadID": squadId,
		})
	} else {
		return squadNotesTmpl.ExecuteWithSession(app, w, r, Values{
			"SquadID": squadId,
		})
	}
}

func (app *App) squadMembersHandler(w http.ResponseWriter, r *http.Request) error {
	params := mux.Vars(r)
	squadId := params["squadId"]

	return squadMembersTmpl.ExecuteWithSession(app, w, r, Values{
		"SquadID": squadId,
	})
}

func (app *App) eventsHandler(w http.ResponseWriter, r *http.Request) error {

	return eventsTmpl.ExecuteWithSession(app, w, r, Values{})
}

func (app *App) eventParticipantsHandler(w http.ResponseWriter, r *http.Request) error {

	params := mux.Vars(r)
	eventId := params["eventId"]

	return participantsTmpl.ExecuteWithSession(app, w, r, Values{"EventID": eventId})
}

func (app *App) homeHandler(w http.ResponseWriter, r *http.Request) error {

	return homeTmpl.ExecuteWithSession(app, w, r, Values{})
}

func (app *App) aboutHandler(w http.ResponseWriter, r *http.Request) error {

	if app.sd.getCurrentUserID(r) != "" {
		return aboutTmpl.ExecuteWithSession(app, w, r, Values{})
	} else {
		return aboutTmpl.Execute(w, nil)
	}
}

func (app *App) userinfoHandler(w http.ResponseWriter, r *http.Request) error {

	u, err := app.sd.getCurrentUserRecord(r)
	if err != nil {
		log.Panic("Failed to get user info: ", err)
		return nil
	}

	currentUserData := app.sd.getCurrentUserData(r)

	currentUserInfo := &struct {
		ContactInfoIssues    bool
		EmailVerified        bool
		DisplayNameNotUnique bool
		Role                 string
		PendingApprove       bool
	}{

		EmailVerified:     u.EmailVerified,
		ContactInfoIssues: !u.EmailVerified || len(currentUserData.DisplayName) == 0,
		Role:              currentUserData.Status.String(),
	}

	if currentUserData.Status == assist_db.PendingApprove {
		currentUserInfo.PendingApprove = true

		users, err := app.db.GetUserByName(r.Context(), currentUserData.DisplayName)
		if users != nil && (len(users) > 1 || len(users) == 1 && users[0] != u.UID) {
			currentUserInfo.DisplayNameNotUnique = true
			currentUserInfo.ContactInfoIssues = true
		}

		if err != nil {
			log.Printf("Got error while checking user name uniqueness: %v", err)
		}

	}

	return userinfoTmpl.ExecuteWithSession(app, w, r, Values{
		"CurrentUserInfo": currentUserInfo})
}

func (app *App) loginHandler(w http.ResponseWriter, r *http.Request) error {
	return loginTmpl.Execute(w, Values{
		"CSRFTag": csrf.TemplateField(r),
		"Session": nil,
	})
}
