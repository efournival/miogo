package main

import (
	"github.com/BurntSushi/toml"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
	"os"
	"reflect"
)

type MiogoConfig struct {
	MongoDBHost     string
	TemporaryFolder string
	SessionDuration int
	CacheDuration   int
}

type Miogo struct {
	db   *MiogoDB
	conf *MiogoConfig
	mux  *http.ServeMux
}

type File struct {
	Name   string        `bson:"name" json:"name"`
	FileID bson.ObjectId `bson:"file_id" json:"-"`
}

type Folder struct {
	Path    string   `bson:"path" json:"path"`
	Files   []File   `bson:"files" json:"files,omitempty"`
	Folders []Folder `json:"folders,omitempty"`
}

type Group struct {
	Id     string `json:"id" bson:"_id,omitempty"`
	Admins []User `json:"admins,omitempty"`
}

type ServiceFunc func(http.ResponseWriter, *http.Request, *User)

func NewMiogo() *Miogo {
	var conf MiogoConfig

	md, err := toml.DecodeFile("miogo.conf", &conf)

	if err != nil {
		log.Fatalf("Error while loading configuration: %s", err)
	}

	s := reflect.ValueOf(&conf).Elem().Type()
	good := true

	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i).Name
		if !md.IsDefined(f) {
			log.Printf("Lacking configuration field: %s\n", f)
			good = false
		}
	}

	if !good {
		log.Fatalf("Please provide the required data in the configuration file")
	}

	os.Setenv("TMPDIR", conf.TemporaryFolder)

	miogo := Miogo{NewMiogoDB(conf.MongoDBHost, conf.CacheDuration, conf.SessionDuration), &conf, http.NewServeMux()}

	miogo.service("GetFile", miogo.GetFile)
	miogo.service("GetFolder", miogo.GetFolder)
	miogo.service("NewFolder", miogo.NewFolder)
	miogo.service("Upload", miogo.Upload)

	miogo.service("Login", miogo.Login)
	// TODO: Logout
	miogo.service("NewUser", miogo.NewUser)
	miogo.service("RemoveUser", miogo.RemoveUser)

	miogo.service("NewGroup", miogo.NewGroup)
	miogo.service("RemoveGroup", miogo.RemoveGroup)
	miogo.service("AddUserToGroup", miogo.AddUserToGroup)
	miogo.service("RemoveUserFromGroup", miogo.RemoveUserFromGroup)
	miogo.service("SetGroupAdmin", miogo.SetGroupAdmin)
	miogo.service("SetResourceRights", miogo.SetResourceRights)

	return &miogo
}

func (m *Miogo) service(name string, sfunc ServiceFunc) {
	m.mux.HandleFunc("/" + name, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			if usr, ok := m.getSessionUser(r); ok || name == "Login" {
				sfunc(w, r, usr)
			} else {
				http.Error(w, "Not logged in", http.StatusForbidden)
			}
		} else {
			http.Error(w, "Please send POST requests", http.StatusBadRequest)
		}
	})
}

