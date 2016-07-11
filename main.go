package main

import (
	"bytes"
	"github.com/BurntSushi/toml"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"fmt"
)

type Miogo struct {
	db     *mgo.Database
	conf   *MiogoConfig
	router *httprouter.Router
}

type MiogoConfig struct {
	MongoDBHost     string
	TemporaryFolder string
}

type File struct {
	Name   string        `bson:"name"`
	FileID bson.ObjectId `bson:"file_id"`
}

type Folder struct {
	ID      bson.ObjectId `bson:"_id,omitempty"`
	Path    string        `bson:"path"`
	Files   []File        `bson:"files"`
	Folders []Folder
}

func NewMiogo() *Miogo {
	var conf MiogoConfig
	if _, err := toml.DecodeFile("miogo.conf", &conf); err != nil {
		log.Fatalf("Error while loading configuration: %s", err)
	}

	os.Setenv("TMPDIR", conf.TemporaryFolder)

	session, err := mgo.Dial(conf.MongoDBHost)
	if err != nil {
		log.Fatalf("Cannot connect to MongoDB: %s", err)
	}

	router := httprouter.New()

	return &Miogo{session.DB("miogo"), &conf, router}
}

func (m *Miogo) View(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	path := strings.TrimRight(ps.ByName("path"), "/")
	if len(path) == 0 {
		path = "/"
	}

	selector := bson.M{"path": path}
	query := m.db.C("folders").Find(selector)

	if count, err := query.Count(); count > 0 && err == nil {
		var folder Folder
		query.One(&folder)

		var subfolders []Folder
		m.db.C("folders").Find(bson.M{"path": bson.RegEx{"^" + path + "/*[^/]+$", ""}}).All(&subfolders)
		folder.Folders = append(folder.Folders, subfolders...)

		t, err := template.ParseFiles("miogo.tpl")

		if err != nil {
			log.Fatalf("Template parsing failure: %s", err)
		}

		t.Execute(w, folder)
		return
	}

	path = strings.TrimRight(path, "/")
	pos := strings.LastIndex(path, "/")
	name := path[pos+1:]
	path = path[:pos]

	if len(path) == 0 {
		path = "/"
	}

	fmt.Printf("Path : " + path + "\n")
	fmt.Printf("Name : " + name + "\n")

	query = m.db.C("folders").Find(bson.M{"path": path, "files.name": name}).Select(bson.M{"files": bson.M{"$elemMatch": bson.M{"name": name}}})

	cnt, _ := query.Count()
	fmt.Printf("Count : %d\n", cnt)

	if count, err := query.Count(); count > 0 && err == nil {
		var folder Folder
		query.One(&folder)

		fmt.Printf("%+v\n", folder)

		file, err := m.db.GridFS("fs").OpenId(folder.Files[0].FileID)

		if err != nil {
			log.Fatalf("Cannot get from GridFS: %s", err)
		}

		io.Copy(w, file)
		file.Close()
		return
	}

	http.Error(w, "Resource does not exist", http.StatusNotFound)
}

func (m *Miogo) NewFolder(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	r.ParseForm()

	name := strings.TrimSpace(r.Form["folderName"][0])

	if strings.ContainsAny(name, "/\\") || len(name) == 0 {
		http.Error(w, "Bad folder name", http.StatusBadRequest)
		return
	}

	path := strings.TrimSpace(r.Form["path"][0])

	if count, err := m.db.C("folders").Find(bson.M{"path": path}).Count(); count == 0 || err != nil {
		http.Error(w, "Path does not exist", http.StatusBadRequest)
		return
	}

	sep := "/"
	if path == sep {
		sep = ""
	}

	m.db.C("folders").Insert(bson.M{"path": path + sep + name})

	http.Redirect(w, r, "/view/" + path, 301)
}

func (m *Miogo) Upload(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	reader, err := r.MultipartReader()

	if err != nil {
		log.Fatalf("Cannot parse multipart form: %s", err)
	}

	files := make(map[bson.ObjectId]string)
	path := "/"

	for {
		part, err := reader.NextPart()

		if err == io.EOF {
			break
		}

		if part.FormName() == "path" {
			buf := new(bytes.Buffer)
			buf.ReadFrom(part)
			path = buf.String()
		} else if part.FormName() == "file" {
			file, err := m.db.GridFS("fs").Create(part.FileName())

			if err != nil {
				log.Fatalf("Cannot create a GridFS file: %s", err)
			}

			io.Copy(file, part)
			file.Close()
			files[file.Id().(bson.ObjectId)] = part.FileName()
		}
	}

	query := bson.M{"path": path}

	if count, err := m.db.C("folders").Find(query).Count(); count > 0 && err == nil {
		// TODO: one update
		for id, name := range files {
			m.db.C("folders").Update(query, bson.M{"$push": bson.M{"files": bson.M{"name": name, "file_id": id}}})
		}
	} else {
		// TODO: remove from GridFS
		http.Error(w, "Path does not exist", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/view/" + path, 301)
}

func main() {
	miogo := NewMiogo()

	query := bson.M{"path": "/"}
	if count, err := miogo.db.C("folders").Find(query).Count(); count == 0 && err == nil {
		miogo.db.C("folders").Insert(query)
	}

	miogo.router.GET("/view/*path", miogo.View)
	miogo.router.POST("/upload", miogo.Upload)
	miogo.router.POST("/newFolder", miogo.NewFolder)

	gracehttp.Serve(&http.Server{Addr: ":8080", Handler: miogo.router})
}
