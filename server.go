package main

import (
	"log"
	"os"
	"reflect"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/valyala/fasthttp"
)

type MiogoConfig struct {
	MongoDBHost     string
	TemporaryFolder string
	SessionDuration int
	AdminEmail      string
	AdminPassword   string
}

type Miogo struct {
	conf            *MiogoConfig
	services        map[string]fasthttp.RequestHandler
	sessionDuration time.Duration
	foldersCache    *Cache
	filesCache      *Cache
	sessionsCache   *Cache
	usersCache      *Cache
	groupsCache     *Cache
}

func (m *Miogo) GetHandler() fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		if f, ok := m.services[string(ctx.Path())]; ok {
			f(ctx)
			return
		}

		ctx.Error("Wrong service name", fasthttp.StatusNotFound)
	}
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

	miogo := Miogo{
		&conf,
		make(map[string]fasthttp.RequestHandler),
		time.Duration(conf.SessionDuration) * time.Minute,
		NewCache(0),
		NewCache(64 >> 20), // 64 megabytes
		NewCache(0),
		NewCache(0),
		NewCache(0),
	}

	miogo.RegisterService(&Service{
		Handler:         miogo.GetFile,
		Options:         NoJSON | NoLoginCheck,
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
	})

	miogo.RegisterService(&Service{
		Handler:         miogo.Login,
		Options:         NoLoginCheck,
		MandatoryFields: []string{"email", "password"},
	})

	miogo.RegisterService(&Service{
		Handler: miogo.Logout,
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
