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

	miogo.RegisterService(&Service{
		Handler:         miogo.GetFile,
		Options:         NoJSON,
		MandatoryFields: []string{"path"},
	})

	miogo.RegisterService(&Service{
		Handler:         miogo.GetFolder,
		MandatoryFields: []string{"path"},
	})

	miogo.RegisterService(&Service{
		Handler:         miogo.NewFolder,
		MandatoryFields: []string{"path"},
	})

	miogo.RegisterService(&Service{
		Handler: miogo.Upload,
		Options: NoFormParsing,
	})

	miogo.RegisterService(&Service{
		Handler:         miogo.Login,
		Options:         NoLoginCheck,
		MandatoryFields: []string{"email", "password"},
	})

	// TODO: Logout

	miogo.RegisterService(&Service{
		Handler:         miogo.NewUser,
		MandatoryFields: []string{"email", "password"},
	})

	miogo.RegisterService(&Service{
		Handler:         miogo.RemoveUser,
		MandatoryFields: []string{"email"},
	})

	miogo.RegisterService(&Service{
		Handler:         miogo.NewGroup,
		MandatoryFields: []string{"group"},
	})

	miogo.RegisterService(&Service{
		Handler:         miogo.RemoveGroup,
		MandatoryFields: []string{"group"},
	})

	miogo.RegisterService(&Service{
		Handler:         miogo.AddUserToGroup,
		MandatoryFields: []string{"user", "group"},
	})

	miogo.RegisterService(&Service{
		Handler:         miogo.RemoveUserFromGroup,
		MandatoryFields: []string{"user", "group"},
	})

	miogo.RegisterService(&Service{
		Handler:         miogo.SetGroupAdmin,
		MandatoryFields: []string{"user", "group"},
	})

	miogo.RegisterService(&Service{
		Handler:         miogo.SetResourceRights,
		MandatoryFields: []string{"resource", "rights"},
		AtLeastOneField: []string{"user", "group", "all"},
	})

	return &miogo
}
