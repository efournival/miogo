package main

import (
	"github.com/BurntSushi/toml"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/mgo.v2/bson"
	"log"
	"os"
)

type MiogoConfig struct {
	MongoDBHost     string
	TemporaryFolder string
}

type Miogo struct {
	db     *MiogoDB
	conf   *MiogoConfig
	router *httprouter.Router
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

type User struct {
	Mail     string  `bson:"mail" json:"mail"`
	Password string  `bson:"password" json:"password"`
	Groups   []Group `json:"groups,omitempty"`
}

type Group struct {
	Id     string `json:"id" bson:"_id,omitempty"`
	Admins []User `json:"admins,omitempty"`
}

func NewMiogo() *Miogo {
	var conf MiogoConfig

	if _, err := toml.DecodeFile("miogo.conf", &conf); err != nil {
		log.Fatalf("Error while loading configuration: %s", err)
	}

	os.Setenv("TMPDIR", conf.TemporaryFolder)

	router := httprouter.New()

	miogo := Miogo{NewMiogoDB(conf.MongoDBHost, 0), &conf, router}

	miogo.router.POST("/GetFile", miogo.GetFile)
	miogo.router.POST("/GetFolder", miogo.GetFolder)
	miogo.router.POST("/NewFolder", miogo.NewFolder)
	miogo.router.POST("/Upload", miogo.Upload)
	miogo.router.POST("/NewUser", miogo.NewUser)
	miogo.router.POST("/RemoveUser", miogo.RemoveUser)
	miogo.router.POST("/NewGroup", miogo.NewGroup)
	miogo.router.POST("/RemoveGroup", miogo.RemoveGroup)
	miogo.router.POST("/AddUserToGroup", miogo.AddUserToGroup)
	miogo.router.POST("/RemoveUserFromGroup", miogo.RemoveUserFromGroup)
	miogo.router.POST("/SetGroupAdmin", miogo.SetGroupAdmin)
	miogo.router.POST("/SetResourceRights", miogo.SetResourceRights)

	return &miogo
}
