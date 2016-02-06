package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	"github.com/mitchellh/cli"
)

// Structs and Vars
const (
	version = "0.1.0"
)

var (
	defaultWorkingDir     = os.Getenv("HOME") + "/.config/shitbucket/"
	defaultDBBind         = "localhost:38080"
	defaultDBName         = "shitbucket"
	defaultDBKeyNamespace = "sb"
)

// Structs
type Meta struct {
	Color bool
	Ui    cli.Ui
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
	Url       string    `json:"url"`
	UrlTitle  string    `json:"url_title"`
	Hash      string    `json:"hash"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
}

type Tag struct {
	Name string   `json:"name"`
	Urls []string `json:"urls"`
}

type Config struct {
	DatabaseBind string `json:"database_addr"`
	DatabaseName string `json:"database_name"`
}

type GlobalContext struct {
	Tags []Tag
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
	if err != nil {
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

// Methods for Url

func (u Url) Uri() string {
	return fmt.Sprintf("/url/%s", u.Hash)
}

func (u Url) DeleteUri() string {
	return fmt.Sprintf("%s/delete", u.Uri())
}

func (u Url) ManageTagsUri() string {
	return fmt.Sprintf("%s/manage-tags", u.Uri())
}

func (u Url) ManageTagsSaveUri() string {
	return fmt.Sprintf("%s/submit", u.ManageTagsUri())
}

func (u Url) HasTags() bool {
	if len(u.Tags) > 0 {
		return true
	}
	return false
}

func (u Url) FormattedCreatedAt() string {
	return u.CreatedAt.Format(time.RFC1123)
}

// Methods for Tag

func (t Tag) HasUrl(url Url) bool {
	for _, h := range t.Urls {
		if h == url.Hash {
			return true
		}
	}
	return false
}

func (t *Tag) RemoveUrl(url Url) {
	for i, h := range t.Urls {
		if h == url.Hash {
			t.Urls = append(t.Urls[:i], t.Urls[i+1:]...)
			saveTag(*t)
		}
	}
}

func (t *Tag) Uri() string {
	return fmt.Sprintf("/tag/%s", t.Name)
}

func (t *Tag) UrlCount() int {
	return len(t.Urls)
}

// Other stuff

func hashUrl(url string) string {
	digest := fmt.Sprintf("%x", md5.Sum([]byte(url)))[:5]
	return digest
}

func makeKeyForUrl(url string) string {
	return fmt.Sprintf("%s:url:%s", defaultDBKeyNamespace, hashUrl(url))
}

func makeKeyForTag(tagname string) string {
	return fmt.Sprintf("%s:tag:%s", defaultDBKeyNamespace, tagname)
}

func buildUrlPath(url string) string {
	path := fmt.Sprintf("http://%s/%s/%s",
		defaultDBBind,
		defaultDBName,
		makeKeyForUrl(url))
	return path
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

func buildTagPath(tagname string) string {
	path := fmt.Sprintf("http://%s/%s/%s",
		defaultDBBind,
		defaultDBName,
		makeKeyForTag(tagname))
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

	r, err := http.Post(buildUrlPath(urlData.Url), "text/json", bytes.NewBuffer(urlBytes))

	if err != nil {
		return err
	}

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

func fetchUrls() ([]Url, error) {
	var urls []Url
	response, err := http.Get(buildPrefixMatchPath("url"))
	if err != nil {
		return []Url{}, err
	}
	scanner := bufio.NewScanner(response.Body)
	defer response.Body.Close()
	for scanner.Scan() {
		url, err := getUrlData(buildPathFromKey(scanner.Text()))
		if err != nil {
			log.Println(err)
		}
		if url.Url != "" {
			urls = append(urls, url)
		}
	}
	return urls, nil
}

func getTags() ([]Tag, error) {
	var tags []Tag
	response, err := http.Get(buildPrefixMatchPath("tag"))
	if err != nil {
		return []Tag{}, err
	}
	scanner := bufio.NewScanner(response.Body)
	defer response.Body.Close()
	for scanner.Scan() {
		tag, err := getTagFromKey(scanner.Text())
		if err != nil {
			log.Println(err)
		}
		if tag.Name != "" {
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

func getTag(path string) (Tag, error) {
	response, err := http.Get(path)
	if err != nil || response.StatusCode == 404 {
		return Tag{}, err
	}

	contents, err := ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	if err != nil {
		return Tag{}, err
	}

	tag := Tag{}
	json.Unmarshal([]byte(contents), &tag)

	return tag, nil
}

func getTagFromKey(key string) (Tag, error) {
	path := buildPathFromKey(key)
	return getTag(path)
}

func getTagFromName(tagname string) (Tag, error) {
	path := buildTagPath(tagname)
	return getTag(path)
}

func getOrCreateTag(tagname string) Tag {
	tag, err := getTagFromName(tagname)
	if tag.Name == "" || err != nil {
		tag.Name = tagname
		saveTag(tag)
	}
	return tag
}

func saveTag(tag Tag) error {
	if tag.Name == "" {
		return errors.New("Can't save a blank record.")
	}
	tagBytes, err := json.Marshal(tag)
	if err != nil {
		return err
	}

	r, _ := http.Post(buildTagPath(tag.Name), "text/json", bytes.NewBuffer(tagBytes))

	if r.StatusCode != 201 {
		return errors.New("Cannot save tag, got status code: " + r.Status)
	}

	return nil
}

// "actions" or whatever you want to call it
func UrlsIndex(rend render.Render) {
	urls, err := fetchUrls()
	if err != nil {
		log.Println(err)
	}
	context := struct {
		Data  map[string]string
		Urls  []Url
		Count int
		Title string
	}{
		Data:  make(map[string]string),
		Urls:  urls,
		Count: len(urls),
		Title: "List of Shit",
	}
	context.Data["ActiveTab"] = "urls"
	rend.HTML(http.StatusOK, "index", context)
}

// "actions" or whatever you want to call it
func APIIndex(rend render.Render) {
	urls, err := fetchUrls()
	if err != nil {
		log.Println(err)
	}
	context := struct {
		Urls  []Url
		Count int
	}{
		Urls:  urls,
		Count: len(urls),
	}
	rend.JSON(http.StatusOK, context)
}

func GetUrl(w http.ResponseWriter, rend render.Render, params martini.Params) {
	urldata, err := getUrlDataFromHash(params["id"])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 Internal Server Error"))
		return
	}
	context := struct {
		Data map[string]string
		Url  Url
	}{
		Data: make(map[string]string),
		Url:  urldata,
	}
	context.Data["ActiveTab"] = "urls"
	rend.HTML(http.StatusOK, "url", context)
}

func AddUrl(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	if urlExists(url) {
		http.Redirect(w, r, "/", http.StatusNotModified)
		return
	}
	title := getPageTitle(url)
	urlRecord := Url{
		Url:       url,
		UrlTitle:  title,
		Hash:      hashUrl(url),
		CreatedAt: time.Now(),
	}
	err := saveUrl(urlRecord)
	if err != nil {
		log.Println(err)
	}
	http.Redirect(w, r, fmt.Sprintf("/url/%s", hashUrl(url)), http.StatusFound)
}

func NewUrl(rend render.Render) {
	context := struct {
		Data map[string]string
	}{
		Data: make(map[string]string),
	}
	context.Data["ActiveTab"] = "urls"
	rend.HTML(http.StatusOK, "add", context)
}

func DeleteUrl(w http.ResponseWriter, r *http.Request, params martini.Params) {
	urldata, err := getUrlDataFromHash(params["id"])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 Internal Server Error"))
		return
	}

	if urldata.Url == "" {
		return
	}

	for _, tag := range urldata.Tags {
		tagObject, err := getTagFromName(tag)
		if err != nil {
			panic(err)
		}
		tagObject.RemoveUrl(urldata)
	}

	client := &http.Client{}
	req, err := http.NewRequest("DELETE", buildUrlPath(urldata.Url), nil)

	resp, err := client.Do(req)

	if err != nil || resp.StatusCode != 200 {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 Internal Server Error"))
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func Auth(res http.ResponseWriter, req *http.Request) {
}

func ManageTags(w http.ResponseWriter, r *http.Request, params martini.Params, rend render.Render) {
	urldata, err := getUrlDataFromHash(params["id"])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 Internal Server Error"))
		return
	}

	if urldata.Url == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	context := struct {
		Data  map[string]string
		Url   Url
		Flash string
		Tags  string
	}{
		Data: make(map[string]string),
		Url:  urldata,
		Tags: strings.Join(urldata.Tags, " "),
	}
	context.Data["ActiveTab"] = "urls"

	rend.HTML(http.StatusOK, "manage-tags", context)
}

func SaveTags(w http.ResponseWriter, r *http.Request, params martini.Params, rend render.Render) {
	urldata, err := getUrlDataFromHash(params["id"])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 Internal Server Error"))
		return
	}

	if urldata.Url == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	tags := r.FormValue("tags")
	//if tags == "" {
	//	context := struct{
	//		Url Url
	//		Flash string
	//		Tags string
	//	} {
	//		Url: urldata,
	//		Flash: "You need to enter some tags, man",
	//		Tags: strings.Join(urldata.Tags, ", "),
	//	}

	//	rend.HTML(http.StatusOK, "manage-tags", context)
	//	return
	//}

	splittags := strings.FieldsFunc(tags, func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c) && !unicode.IsPunct(c)
	})

	// Reset Tags
	urldata.Tags = []string{}
	ts, err := getTags()
	for _, tag := range ts {
		tag.RemoveUrl(urldata)
	}

	for _, tagname := range splittags {
		tag := getOrCreateTag(tagname)
		tag.Urls = append(tag.Urls, urldata.Hash)
		urldata.Tags = append(urldata.Tags, tag.Name)
		saveTag(tag)
	}

	saveUrl(urldata)

	http.Redirect(w, r, urldata.Uri(), http.StatusFound)
}

func TagIndex(w http.ResponseWriter, r *http.Request, params martini.Params, rend render.Render) {
	tags, err := getTags()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 Internal Server Error"))
		return
	}

	context := struct {
		Data      map[string]string
		ActiveTab string
		Tags      []Tag
		Count     int
	}{
		Data:      make(map[string]string),
		ActiveTab: "tags",
		Tags:      tags,
		Count:     len(tags),
	}
	context.Data["ActiveTab"] = "tags"

	rend.HTML(http.StatusOK, "tag-index", context)
}

func GetTag(w http.ResponseWriter, r *http.Request, params martini.Params, rend render.Render) {
	tag, err := getTagFromName(params["id"])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 Internal Server Error"))
		return
	}

	if tag.Name == "" {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("OH SHIT A 404"))
		return
	}

	var urls []Url

	for _, h := range tag.Urls {
		url, err := getUrlDataFromHash(h)
		if url.Url != "" && err == nil {
			urls = append(urls, url)
		}
	}

	context := struct {
		Data  map[string]string
		Urls  []Url
		Count int
		Title string
		Tag   Tag
	}{
		Data:  make(map[string]string),
		Urls:  urls,
		Count: len(tag.Urls),
		Title: fmt.Sprintf("Urls Tagged Under %s", tag.Name),
		Tag:   tag,
	}
	context.Data["ActiveTab"] = "tags"

	rend.HTML(http.StatusOK, "index", context)
}

// Main shit

func wrappedrun(bind string) error {
	m := martini.Classic()
	m.Use(Auth)
	m.Use(martini.Static("assets"))
	m.Use(render.Renderer(render.Options{
		Directory:  "templates",
		Extensions: []string{".html"},
		Charset:    "UTF-8",
		Layout:     "base",
	}))

	// Routes
	m.Get("/", UrlsIndex)

	m.Group("/url", func(r martini.Router) {
		r.Get("/new", NewUrl)
		r.Post("/submit", AddUrl)
		r.Get("/:id/delete", DeleteUrl)
		r.Get("/:id/manage-tags", ManageTags)
		r.Post("/:id/manage-tags/submit", SaveTags)
		r.Get("/:id", GetUrl)
	})

	m.Get("/tag", TagIndex)
	m.Group("/tag", func(r martini.Router) {
		r.Get("/:id", GetTag)
	})

	m.Get("/api/urls", APIIndex)

	return http.ListenAndServe(bind, m)
}

func main() {
	ui := &cli.BasicUi{Writer: os.Stdout}

	meta := Meta{
		Color: true,
		Ui:    ui,
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
