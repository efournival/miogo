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

	return &miogo
}
