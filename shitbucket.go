package main

import (
	"net/http"
	"fmt"
	"log"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
)

func GetUrls() string {
	return "hello"
}

func GetUrl(params martini.Params) string {
	return fmt.Sprintf("sup %s", params["id"])
}

func NewUrl() string {
	return "sup submit"
}

func UpdateUrl(params martini.Params) string {
	return fmt.Sprintf("update %s", params["id"])
}

func DeleteUrl(params martini.Params) string {
	return fmt.Sprintf("delete %s", params["id"])
}

func Auth(res http.ResponseWriter, req *http.Request) {

}

func main() {
	m := martini.Classic()
	m.Use(Auth)
	m.Use(render.Renderer(render.Options{
		Directory: "templates",
		Extentions: []string{".html"},
		Charset: "UTF-8",
	}))

	// Routes
	m.Get("/", GetUrls)

	m.Group("/url", func(r martini.Router) {
		r.Get("/:id", GetUrl)
		r.Post("/submit", NewUrl)
		r.Put("/update/:id", UpdateUrl)
		r.Delete("/delete/:id", DeleteUrl)
	})

	log.Fatal(http.ListenAndServe(":8080", m))
}
