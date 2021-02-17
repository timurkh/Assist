package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"text/template"

	"github.com/gorilla/csrf"
)

type Values map[string]interface{}

func parseTemplate(filename string) *appTemplate {
	path := filepath.Join("templates", filename)
	tmpl := template.Must(template.ParseFiles(path))

	return &appTemplate{*tmpl}
}

// parseBodyTemplate applies a given file to the body of the base template.
func parseBodyTemplate(filename string) *appTemplate {
	tmpl := template.Must(template.ParseFiles("templates/base.html"))

	// Put the named file into a template called "body"
	path := filepath.Join("templates", filename)
	b, err := ioutil.ReadFile(path)
	if err != nil {
		panic(fmt.Errorf("could not read template: %v", err))
	}
	template.Must(tmpl.New("body").Parse(string(b)))

	return &appTemplate{*tmpl.Lookup("base.html")}
}

type appTemplate struct {
	template.Template
}

// Execute writes the template using the provided data.
func (tmpl *appTemplate) ExecuteWithSession(app *App, w http.ResponseWriter, r *http.Request, values Values) error {

	values["Session"] = app.su.getSessionData(r)
	values["CSRFTag"] = csrf.TemplateField(r)

	if err := tmpl.Execute(w, values); err != nil {
		log.Panicf("could not write template: %v", err)
		return err
	}
	return nil
}
