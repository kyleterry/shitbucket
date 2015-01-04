package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"flag"
	"fmt"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"github.com/mitchellh/cli"
	"github.com/PuerkitoBio/goquery"
)

// Structs and Vars
const (
	version = "0.1.0"
)

var (
	defaultWorkingDir = os.Getenv("HOME") + "/.config/shitbucket/"
	defaultDBBind = "localhost:38080"
	defaultDBName = "shitbucket"
	defaultDBKeyNamespace = "sb"
)

// Structs
type Meta struct {
	Color bool
	Ui cli.Ui
}

type AdminCommand struct {
	Meta
}

type SetupCommand struct {
	Meta
}

type RunCommand struct {
	Meta
}

type Url struct {
	Url string `json:"url"`
	UrlTitle string `json:"url_title"`
	CreatedAt time.Time `json:"created_at"`
}

type Tag struct {
	Name string `json:"name"`
	Entries []string `json:"entries"`
}

type Config struct {
	DatabaseBind string `json:"database_addr"`
	DatabaseName string `json:"database_name"`
}

// Methods for commands
func (c *AdminCommand) Run(args []string) int {
	return 1
}

func (c *AdminCommand) Help() string {
	return "Not implemented yet"
}

func (c *AdminCommand) Synopsis() string {
	return "Not implemented yet"
}

func (c *SetupCommand) Run(args []string) int {
	os.Mkdir(defaultWorkingDir, 0700)
	return 0
}

func (c *SetupCommand) Synopsis() string {

	return "Sets up shitbucket for the fist time"
}

func (c *SetupCommand) Help() string {

	return "Not implemented yet"
}

func (c *RunCommand) Run(args []string) int {
	var bind string
	cmdFlags := flag.NewFlagSet("Run", flag.ContinueOnError)
	cmdFlags.StringVar(&bind, "bind", "localhost:8080", "bind")
	cmdFlags.StringVar(&defaultDBBind, "db-bind", defaultDBBind, "db-bind")
	cmdFlags.StringVar(&defaultDBName, "db-name", defaultDBName, "db-name")
	cmdFlags.StringVar(&defaultDBKeyNamespace, "db-key-namespace", defaultDBKeyNamespace, "db-key-namespace")
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}
	log.Printf("Listening on http://%s", bind)
	log.Printf("DB location http://%s/%s", defaultDBBind, defaultDBName)

	err := wrappedrun(bind)
	if err != nil{
		return 1
	}
	return 0
}

func (c *RunCommand) Synopsis() string {
	return "Starts the shitbucket web server"
}

func (c *RunCommand) Help() string {
	help := `
Usage: shitbucket run [options]
	Start a shitbucket web server.

Options:
	-bind				Set how the server should bind. E.g. -bind "localhost:9090"
						Default is ":8080".

	-db-bind			This is the address (tcp) where the database listens.
						"localhost:38080" is used by default.

	-db-name			Sets the name to use for the database namespace.
						"shitbucket" is used by default.

	-db-key-namespace	Sets the namespace to use in the database when storing shit.
						"sb" is the default.
`
	return help
}

// Other stuff

func hashUrl(url string) string {
	digest := fmt.Sprintf("%x", md5.Sum([]byte(url)))[:5]
	fmt.Println(digest)
	return digest
}

func buildUrlPath(url string) string {
	path := fmt.Sprintf("http://%s/%s/%s:url:%s",
						defaultDBBind,
						defaultDBName,
						defaultDBKeyNamespace,
						hashUrl(url))
	fmt.Println(path)
	return path
}

func buildSubmitPath() string {
	return fmt.Sprintf("http://%s/%s/url/submit",
						defaultDBBind,
						defaultDBName)
}

func buildPrefixMatchPath(prefix string) string {
	path := fmt.Sprintf("http://%s/%s/%s:%s/_match",
						defaultDBBind,
						defaultDBName,
						defaultDBKeyNamespace,
						prefix)
	return path
}

func buildPathFromKey(key string) string {
	path := fmt.Sprintf("http://%s/%s/%s",
						defaultDBBind,
						defaultDBName,
						key)
	return path
}

func getUrlData(urlpath string) (Url, error) {
	urldata := Url{}
	response, err := http.Get(urlpath)
	if err != nil || response.StatusCode == 404 {
		return urldata, err
	}
	contents, err := ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	if err != nil {
		return urldata, err
	}
	json.Unmarshal([]byte(contents), &urldata)
	return urldata, nil
}

func getUrlDataFromUrl(url string) (Url, error) {
	urlpath := buildUrlPath(url)
	urldata, err := getUrlData(urlpath)
	return urldata, err
}

func getUrlDataFromHash(hash string) (Url, error) {
	path := buildPathFromKey(fmt.Sprintf("%s:url:%s", defaultDBKeyNamespace, hash))
	urldata, err := getUrlData(path)
	return urldata, err
}

func urlExists(url string) bool {
	urldata, err := getUrlData(url)
	if err != nil {
		log.Println(err)
		return false
	}
	if urldata.Url == "" {
		return false
	}
	return true
}


func saveUrl(urlData Url) error {
	if urlData.Url == "" {
		return errors.New("Can't save a blank record.")
	}
	urlBytes, err := json.Marshal(urlData)
	if err != nil {
		return err
	}

	r, _ := http.Post(buildUrlPath(urlData.Url), "text/json", bytes.NewBuffer(urlBytes))

	if r.StatusCode != 201 {
		return errors.New("Cannot save url, got status code: " + r.Status)
	}

	return nil
}

func getPageTitle(url string) string {
	doc, err := goquery.NewDocument(url)
	if err != nil {
		return ""
	}

	title := doc.Find("title").Text()
	return title
}

func fetchUrls() ([]*Url, error) {
	var urls []*Url
	response, err := http.Get(buildPrefixMatchPath("url"))
	if err != nil {
		return []*Url{}, err
	}
	scanner := bufio.NewScanner(response.Body)
	defer response.Body.Close()
	for scanner.Scan() {
		url, err := getUrlData(buildPathFromKey(scanner.Text()))
		if err != nil {
			log.Println(err)
		}
		urls = append(urls, &url)
	}
	return urls, nil
}

// "actions" or whatever you want to call it
func GetUrls(rend render.Render) {
	urls, err := fetchUrls()
	if err != nil {
		log.Println(err)
	}
	urldata := struct {
		Urls []*Url
	} {
		Urls: urls,
	}
	rend.HTML(http.StatusOK, "index", urldata)
}

func GetUrl(w http.ResponseWriter, rend render.Render, params martini.Params) {
	urldata, err := getUrlDataFromHash(params["id"])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 Internal Server Error"))
		return
	}
	rend.HTML(http.StatusOK, "url", urldata)
}

func AddUrl(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	if urlExists(url) {
		http.Redirect(w, r, "/", http.StatusNotModified)
		return
	}
	title := getPageTitle(url)
	urlRecord := Url{
		Url: url,
		UrlTitle: title,
		CreatedAt: time.Now(),
	}
	log.Println(title)
	err := saveUrl(urlRecord)
	if err != nil {
		log.Println(err)
	}
	http.Redirect(w, r, fmt.Sprintf("/url/%s", hashUrl(url)), http.StatusFound)
}

func NewUrl(rend render.Render) {
	rend.HTML(http.StatusOK, "add", nil)
}

func UpdateUrl(params martini.Params) string {
	return fmt.Sprintf("update %s", params["id"])
}

func DeleteUrl(params martini.Params) string {
	return fmt.Sprintf("delete %s", params["id"])
}

func Auth(res http.ResponseWriter, req *http.Request) {

}

func wrappedrun(bind string) error {
	m := martini.Classic()
	m.Use(Auth)
	m.Use(martini.Static("assets"))
	m.Use(render.Renderer(render.Options{
		Directory: "templates",
		Extensions: []string{".html"},
		Charset: "UTF-8",
		Layout: "base",
	}))

	// Routes
	m.Get("/", GetUrls)

	m.Group("/url", func(r martini.Router) {
		r.Get("/new", NewUrl)
		r.Post("/submit", AddUrl)
		r.Put("/update/:id", UpdateUrl)
		r.Delete("/delete/:id", DeleteUrl)
		r.Get("/:id", GetUrl)
	})

	return http.ListenAndServe(bind, m)
}

func main() {
	ui := &cli.BasicUi{Writer: os.Stdout}

	meta := Meta{
		Color: true,
		Ui: ui,
	}

	c := cli.NewCLI("shitbucket", version)
	c.Args = os.Args[1:]
	c.Commands = map[string]cli.CommandFactory{
		"admin": func() (cli.Command, error) {
			return &AdminCommand{
				Meta: meta,
			}, nil
		},
		"setup": func() (cli.Command, error) {
			return &SetupCommand{
				Meta: meta,
			}, nil
		},
		"run": func() (cli.Command, error) {
			return &RunCommand{
				Meta: meta,
			}, nil
		},
	}

	exitCode, err := c.Run()
	if err != nil {
		ui.Error(fmt.Sprintf("Error executing CLI: %s", err.Error()))
		os.Exit(1)
	}
	os.Exit(exitCode)
}
