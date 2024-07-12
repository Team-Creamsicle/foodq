package main

import (
	"fmt"
	"net/http"

	"github.com/a-h/templ"
	"github.com/team-creamsicle/foodq/frontend/templates"
)

func main() {
	component := templates.Hello("John")

	http.Handle("/", templ.Handler(component))

	fmt.Println("Listening on :3000")
	http.ListenAndServe(":3000", nil)
}
