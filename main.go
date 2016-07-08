package main

import (
	"bytes"
	"github.com/BurntSushi/toml"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"log"
	"net/http"
	"os"
	"html/template"
	"strings"
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
	ID    bson.ObjectId `bson:"_id,omitempty"`
	Path  string        `bson:"path"`
	Files []File        `bson:"files"`
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
	path := "/"

	if len(ps.ByName("path")) > 0 {
		path = ps.ByName("path")

		if path[:len(path)-1] == "/" {
			path = path[len(path)-1:]
		}
	}

	selector := bson.M{"path": path}
	query := m.db.C("folders").Find(selector)

	if count, err := query.Count(); count > 0 && err == nil {
		var folder Folder
		query.One(&folder)
		t, err := template.ParseFiles("miogo.tpl")

		if err != nil {
			log.Fatalf("Template parsing failure: %s", err)
		}

		t.Execute(w, folder)
		return
	}

	pos := strings.LastIndex(path, "/")
	var name string

	if pos == -1 {
		name = path
		path = "/"
	} else {
		name = path[pos+1:]
		path = path[:pos+1]
	}

	query = m.db.C("folders").Find(nil).Select(bson.M{"path": path, "files": bson.M{"$elemMatch": bson.M{"name": name}}})

	if count, err := query.Count(); count > 0 && err == nil {
		var folder Folder
		query.One(&folder)

		file, err := m.db.GridFS("fs").OpenId(folder.Files[0].FileID)

		if err != nil {
			log.Fatalf("Cannot get from GridFS: %s", err)
		}

		io.Copy(w, file)
		file.Close()
		return
	}

	// error

}

func (m *Miogo) NewFile(path, name string, fileID bson.ObjectId) {
	query := bson.M{"path": path}

	if count, err := m.db.C("folders").Find(query).Count(); count > 0 && err == nil {
		m.db.C("folders").Update(query, bson.M{"$push": bson.M{"files": bson.M{"name": name, "file_id": fileID}}})
	}
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

	for id, name := range files {
		m.NewFile(path, name, id)
	}
}

func main() {
	miogo := NewMiogo()

	miogo.router.GET("/view/*path", miogo.View)
	miogo.router.POST("/upload", miogo.Upload)

	gracehttp.Serve(&http.Server{Addr: ":8080", Handler: miogo.router})
}
