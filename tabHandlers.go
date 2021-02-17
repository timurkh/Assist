package main

import (
	"assist/db"
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

	return squadsTmpl.Execute(app, w, r, struct {
		Session *SessionData
		CSRFTag template.HTML
	}{app.su.getSessionData(r), csrf.TemplateField(r)})
}

func (app *App) squadDetailsHandler(w http.ResponseWriter, r *http.Request) error {
	params := mux.Vars(r)
	squadId := params["squadId"]

	_, level := app.checkAuthorization(r, "me", squadId, squadMember|squadAdmin|squadOwner|systemAdmin)

	if level >= squadAdmin {
		return squadDetailsTmpl.Execute(app, w, r, struct {
			Session *SessionData
			SquadID string
			CSRFTag template.HTML
		}{app.su.getSessionData(r), squadId, csrf.TemplateField(r)})
	} else {
		return squadNotesTmpl.Execute(app, w, r, struct {
			Session *SessionData
			SquadID string
			CSRFTag template.HTML
		}{app.su.getSessionData(r), squadId, csrf.TemplateField(r)})
	}
}

func (app *App) squadMembersHandler(w http.ResponseWriter, r *http.Request) error {
	params := mux.Vars(r)
	squadId := params["squadId"]

	return squadMembersTmpl.Execute(app, w, r, struct {
		Session *SessionData
		SquadID string
		CSRFTag template.HTML
	}{app.su.getSessionData(r), squadId, csrf.TemplateField(r)})
}

func (app *App) eventsHandler(w http.ResponseWriter, r *http.Request) error {

	return eventsTmpl.Execute(app, w, r, struct {
		Session *SessionData
		CSRFTag template.HTML
	}{app.su.getSessionData(r), csrf.TemplateField(r)})
}

func (app *App) homeHandler(w http.ResponseWriter, r *http.Request) error {

	return homeTmpl.Execute(app, w, r, struct {
		Session *SessionData
		CSRFTag template.HTML
	}{app.su.getSessionData(r), csrf.TemplateField(r)})
}

func (app *App) aboutHandler(w http.ResponseWriter, r *http.Request) error {

	if app.su.getCurrentUserID(r) != "" {
		return aboutTmpl.Execute(app, w, r, struct {
			Session *SessionData
			CSRFTag template.HTML
		}{app.su.getSessionData(r), csrf.TemplateField(r)})
	} else {
		return aboutTmpl.Execute(app, w, r, struct {
			Session *SessionData
			CSRFTag template.HTML
		}{nil, csrf.TemplateField(r)})
	}
}

func (app *App) userinfoHandler(w http.ResponseWriter, r *http.Request) error {

	ctx := r.Context()
	sessionData := app.su.getSessionData(r)
	user, _ := app.db.GetUser(ctx, sessionData.UID)

	return userinfoTmpl.Execute(app, w, r, struct {
		Session *SessionData
		Data    *db.UserInfo
		CSRFTag template.HTML
	}{app.su.getSessionData(r), user, csrf.TemplateField(r)})
}

func (app *App) loginHandler(w http.ResponseWriter, r *http.Request) error {
	return loginTmpl.Execute(app, w, r, struct {
		Session *SessionData
		CSRFTag template.HTML
	}{nil, csrf.TemplateField(r)})
}
