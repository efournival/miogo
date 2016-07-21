package main

import (
	"log"
	"net/http"
	"os"
	"reflect"
	"time"

	"github.com/BurntSushi/toml"
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
	conf            *MiogoConfig
	mux             *http.ServeMux
	sessionDuration time.Duration
	filesCache      *Cache
	foldersCache    *Cache
	sessionsCache   *Cache
	usersCache      *Cache
	groupsCache     *Cache
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

	InitDB(conf.MongoDBHost, conf.AdminEmail, conf.AdminPassword)

	dur := time.Duration(conf.CacheDuration) * time.Minute

	miogo := Miogo{
		&conf,
		http.NewServeMux(),
		time.Duration(conf.SessionDuration) * time.Minute,
		NewCache(dur),
		NewCache(dur),
		NewCache(dur),
		NewCache(dur),
		NewCache(dur),
	}

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

	miogo.RegisterService(&Service{
		Handler:         miogo.Logout,
		Options:         NoFormParsing,
		MandatoryFields: []string{"path"},
	})

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
		MandatoryFields: []string{"name"},
	})

	miogo.RegisterService(&Service{
		Handler:         miogo.RemoveGroup,
		MandatoryFields: []string{"name"},
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
