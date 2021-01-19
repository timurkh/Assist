package main

import (
	"context"
	"fmt"
	"net/http"
)

var (
	firebaseTmpl = parseTemplate("firebase.html")
	loginTmpl    = parseTemplate("login.html")
	homeTmpl     = parseBodyTemplate("home.html")
	userinfoTmpl = parseBodyTemplate("userinfo.html")
	squadsTmpl   = parseBodyTemplate("squads.html")
)

func (app *App) squadsHandler(w http.ResponseWriter, r *http.Request) error {

	return squadsTmpl.Execute(app, w, r, struct {
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

func (app *App) userinfoHandler(w http.ResponseWriter, r *http.Request) error {

	ctx := context.Background()
	users := app.db.getUsersDatabase()

	sessionData := app.getSessionData(r)
	user, _ := users.GetUserDetails(ctx, sessionData.UID)

	return userinfoTmpl.Execute(app, w, r, struct {
		Session *SessionData
		Data    *UserDetails
	}{sessionData, user})
}

func (app *App) loginHandler(w http.ResponseWriter, r *http.Request) error {
	return loginTmpl.Execute(app, w, r, nil)
}
