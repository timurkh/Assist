package main

import (
	"fmt"
	"log"
	"net/http"
)

var (
	firebaseTmpl = parseTemplate("firebase.html")
	loginTmpl    = parseTemplate("login.html")
	homeTmpl     = parseBodyTemplate("home.html")
	userinfoTmpl = parseBodyTemplate("userinfo.html")
	squadsTmpl   = parseBodyTemplate("squads.html")
	squadTmpl    = parseBodyTemplate("squad.html")
	eventsTmpl   = parseBodyTemplate("events.html")
	aboutTmpl    = parseBodyTemplate("about.html")
)

func (app *App) squadsHandler(w http.ResponseWriter, r *http.Request) error {

	return squadsTmpl.Execute(app, w, r, struct {
		Session *SessionData
	}{app.getSessionData(r)})
}

func (app *App) squadHandler(w http.ResponseWriter, r *http.Request) error {
	keys, ok := r.URL.Query()["squadId"]

	if !ok || len(keys[0]) < 1 {
		err := fmt.Errorf("Missing Squad ID")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	// Query()["key"] will return an array of items,
	// we only want the single item.
	squadId := keys[0]

	ctx := r.Context()
	squadInfo, err := app.dbSquads.GetSquad(ctx, squadId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return err
	}

	userId := "me"
	if app.checkAuthorization(r, &userId, squadInfo, squadOwner|squadMember) == 0 {
		err = fmt.Errorf("Current user is not authorized to get squad %v info", squadId)
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return err
	}

	return squadTmpl.Execute(app, w, r, struct {
		Session *SessionData
		Squad   *SquadInfo
	}{app.getSessionData(r), squadInfo})
}

func (app *App) eventsHandler(w http.ResponseWriter, r *http.Request) error {

	return eventsTmpl.Execute(app, w, r, struct {
		Session *SessionData
	}{app.getSessionData(r)})
}

func (app *App) homeHandler(w http.ResponseWriter, r *http.Request) error {

	u, _ := app.getCurrentUserInfo(r)
	return homeTmpl.Execute(app, w, r, struct {
		Session *SessionData
		Data    string
	}{
		app.getSessionData(r),
		fmt.Sprintf("%+v<br>%+v", u, u.ProviderUserInfo[0])})
}

func (app *App) aboutHandler(w http.ResponseWriter, r *http.Request) error {

	return aboutTmpl.Execute(app, w, r, struct {
		Session *SessionData
	}{
		app.getSessionData(r),
	})
}

func (app *App) userinfoHandler(w http.ResponseWriter, r *http.Request) error {

	ctx := r.Context()
	sessionData := app.getSessionData(r)
	user, _ := app.dbUsers.GetUser(ctx, sessionData.UID)

	return userinfoTmpl.Execute(app, w, r, struct {
		Session *SessionData
		Data    *UserInfo
	}{sessionData, user})
}

func (app *App) loginHandler(w http.ResponseWriter, r *http.Request) error {
	return loginTmpl.Execute(app, w, r, nil)
}
