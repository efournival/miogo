package main

import (
	"log"
	"net/http"
	"os"
	"reflect"

	"github.com/BurntSushi/toml"
	"gopkg.in/mgo.v2/bson"
)

type MiogoConfig struct {
	MongoDBHost     string
	TemporaryFolder string
	SessionDuration int
	CacheDuration   int
	AdminEmail      string
	AdminPassword   string
}

type Miogo struct {
	db   *MiogoDB
	conf *MiogoConfig
	mux  *http.ServeMux
}

type File struct {
	Name   string        `bson:"name" json:"name"`
	FileID bson.ObjectId `bson:"file_id" json:"-"`
	Rights Right         `json:"rights,omitempty" bson:"rights"`
}

type Folder struct {
	Path    string   `bson:"path" json:"path"`
	Files   []File   `bson:"files" json:"files,omitempty"`
	Folders []Folder `json:"folders,omitempty"`
	Rights  Right    `json:"rights,omitempty" bson:"rights"`
}

type Right struct {
	All    string        `json:"all,omitempty" bson:"rights"`
	Groups []EntityRight `json:"groups,omitempty" bson:"groups"`
	Users  []EntityRight `json:"users,omitempty" bson:"users"`
}

type EntityRight struct {
	Name   string `json:"name,omitempty" bson:"name"`
	Rights string `json:"rights,omitempty" bson:"rights"`
}

type Group struct {
	Id     string `json:"id" bson:"_id,omitempty"`
	Admins []User `json:"admins,omitempty"`
}

type ServiceFunc func(http.ResponseWriter, *http.Request, *User)

type MW func(ServiceFunc, *Miogo) ServiceFunc

var JSON = func(sfunc ServiceFunc, _ *Miogo) ServiceFunc {
	return func(w http.ResponseWriter, r *http.Request, u *User) {
		w.Header().Set("Content-Type", "application/json")
		sfunc(w, r, u)
	}
}

var Args = func(sfunc ServiceFunc, _ *Miogo) ServiceFunc {
	return func(w http.ResponseWriter, r *http.Request, u *User) {
		r.ParseForm()
		sfunc(w, r, u)
	}
}

var Logged = func(sfunc ServiceFunc, m *Miogo) ServiceFunc {
	return func(w http.ResponseWriter, r *http.Request, _ *User) {
		if u, ok := m.getSessionUser(r); ok {
			sfunc(w, r, u)
		} else {
			http.Error(w, "Not logged in", http.StatusForbidden)
		}
	}
}

var POST = func(sfunc ServiceFunc, _ *Miogo) ServiceFunc {
	return func(w http.ResponseWriter, r *http.Request, u *User) {
		if r.Method == "POST" {
			sfunc(w, r, u)
		} else {
			http.Error(w, "Please send POST requests", http.StatusBadRequest)
		}
	}
}

func (m *Miogo) service(name string, sfunc ServiceFunc, checks []MW) {
	for _, f := range checks {
		sfunc = f(sfunc, m)
	}

	m.mux.HandleFunc("/"+name, func(w http.ResponseWriter, r *http.Request) {
		sfunc(w, r, nil)
	})
}

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

	miogo := Miogo{NewMiogoDB(conf.MongoDBHost, conf.CacheDuration, conf.SessionDuration, conf.AdminEmail, conf.AdminPassword), &conf, http.NewServeMux()}

	miogo.service("GetFile", miogo.GetFile, []MW{POST, Logged, JSON, Args})
	miogo.service("GetFolder", miogo.GetFolder, []MW{POST, Logged, JSON, Args})
	miogo.service("NewFolder", miogo.NewFolder, []MW{POST, Logged, JSON, Args})
	miogo.service("Upload", miogo.Upload, []MW{Logged, JSON, Args})

	miogo.service("Login", miogo.Login, []MW{POST, JSON, Args})
	// TODO: Logout
	miogo.service("NewUser", miogo.NewUser, []MW{POST, Logged, JSON, Args})
	miogo.service("RemoveUser", miogo.RemoveUser, []MW{POST, Logged, JSON, Args})

	miogo.service("NewGroup", miogo.NewGroup, []MW{POST, Logged, JSON, Args})
	miogo.service("RemoveGroup", miogo.RemoveGroup, []MW{POST, Logged, JSON, Args})
	miogo.service("AddUserToGroup", miogo.AddUserToGroup, []MW{POST, Logged, JSON, Args})
	miogo.service("RemoveUserFromGroup", miogo.RemoveUserFromGroup, []MW{POST, Logged, JSON, Args})
	miogo.service("SetGroupAdmin", miogo.SetGroupAdmin, []MW{POST, Logged, JSON, Args})
	miogo.service("SetResourceRights", miogo.SetResourceRights, []MW{POST, Logged, JSON, Args})

	return &miogo
}
