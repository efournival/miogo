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

type Right struct {
	All    string        `bson:"all" json:"all,omitempty"`
	Groups []EntityRight `bson:"groups" json:"groups,omitempty"`
	Users  []EntityRight `bson:"users" json:"users,omitempty"`
}

type EntityRight struct {
	Name   string `bson:"name" json:"name,omitempty"`
	Rights string `bson:"rights" json:"rights,omitempty"`
}

type File struct {
	Name   string        `bson:"name" json:"name"`
	FileID bson.ObjectId `bson:"file_id" json:"-"`
	Rights *Right        `bson:"rights,omitempty" json:"rights,omitempty"`
}

type Folder struct {
	Path    string   `bson:"path" json:"path"`
	Files   []File   `bson:"files" json:"files,omitempty"`
	Folders []Folder `json:"folders,omitempty"`
	Rights  *Right   `bson:"rights,omitempty" json:"rights,omitempty"`
}

type Group struct {
	Id     string `bson:"_id,omitempty" json:"id"`
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
