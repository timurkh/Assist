package main

import (
	"html/template"
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
	aboutTmpl        = parseBodyTemplate("about.html")
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

func (app *App) homeHandler(w http.ResponseWriter, r *http.Request) error {

	return homeTmpl.ExecuteWithSession(app, w, r, Values{})
}

func (app *App) aboutHandler(w http.ResponseWriter, r *http.Request) error {

	if app.su.getCurrentUserID(r) != "" {
		return aboutTmpl.ExecuteWithSession(app, w, r, Values{})
	} else {
		return aboutTmpl.Execute(w, nil)
	}
}

func (app *App) userinfoHandler(w http.ResponseWriter, r *http.Request) error {

	return userinfoTmpl.ExecuteWithSession(app, w, r, Values{})
}

func (app *App) loginHandler(w http.ResponseWriter, r *http.Request) error {
	return loginTmpl.Execute(w, struct {
		CSRFTag template.HTML
	}{csrf.TemplateField(r)})
}
