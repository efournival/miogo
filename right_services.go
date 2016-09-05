package main

import (
	"errors"
	"log"
	"strings"

	"github.com/valyala/fasthttp"

	"gopkg.in/mgo.v2/bson"
)

type Right struct {
	All    string        `bson:"all" json:"all,omitempty"`
	Groups []EntityRight `bson:"groups" json:"groups,omitempty"`
	Users  []EntityRight `bson:"users" json:"users,omitempty"`
}

type EntityRight struct {
	Name   string `bson:"name" json:"name,omitempty"`
	Rights string `bson:"rights" json:"rights,omitempty"`
}

func (m *Miogo) SetResourceRightsP(resource, rights, entityType, name string) error {
	var err error

	if folder, ok := m.FetchFolder(resource); ok {
		selector := bson.M{"path": bson.M{"$regex": bson.RegEx{`^` + resource, ""}}}

		for _, childFile := range folder.Files {
			err = m.SetResourceRightsP(resource+"/"+childFile.Name, rights, entityType, name)
			if err != nil {
				return errors.New("Can't set rights for child file")
			}
		}

		for _, childFolder := range folder.Folders {
			err = m.SetResourceRightsP(childFolder.Path, rights, entityType, name)
			if err != nil {
				return errors.New("Can't set rights for child folder")
			}
		}

		if entityType == "all" {
			_, err = db.C("folders").UpdateAll(selector, bson.M{"$set": bson.M{"rights.all": rights}})
		} else {
			err = db.C("folders").Update(selector, bson.M{"$addToSet": bson.M{"rights." + entityType: bson.M{"name": name, "rights": rights}}})
		}

		if err != nil {
			return errors.New("Can't set rights for folder")
		}
		m.foldersCache.InvalidateStartWith(resource)

	} else if m.FileExists(resource) {
		dir, file := formatF(resource)
		log.Println(dir)
		log.Println(file)

		if entityType == "all" {
			err = db.C("folders").Update(
				bson.M{"path": dir, "files.name": file},
				bson.M{"$set": bson.M{"files.0.rights.all": rights}})
		} else {
			err = db.C("folders").Update(
				bson.M{"path": dir, "files.name": file},
				bson.M{"$addToSet": bson.M{"files.0.rights." + entityType: bson.M{"name": name, "rights": rights}}})
		}

		if err != nil {
			return errors.New("Can't set rights for file")
		}
	}
	return nil
}

func (m *Miogo) SetResourceRights(ctx *fasthttp.RequestCtx, u *User) {
	rights := strings.TrimSpace(string(ctx.FormValue("rights")))
	resource := formatD(string(ctx.FormValue("resource")))

	var (
		name, entityType string
	)

	if len(ctx.FormValue("user")) > 0 {
		name = strings.TrimSpace(string(ctx.FormValue("user")))
		entityType = "users"
	} else if len(ctx.FormValue("group")) > 0 {
		name = strings.TrimSpace(string(ctx.FormValue("group")))
		entityType = "groups"
	} else {
		entityType = "all"
	}

	var err error

	if _, ok := m.FetchFolder(resource); ok {
		m.SetResourceRightsP(resource, rights, entityType, name)
	} else if m.FileExists(resource) {
		err = m.SetResourceRightsP(resource, rights, entityType, name)
		if err != nil {
			ctx.SetBodyString(`{ "error": "Cannot set file rights" }`)
			return
		}
		dir, _ := formatF(resource)
		m.foldersCache.Invalidate(dir)
	} else {
		ctx.SetBodyString(`{ "error": "Resource does not exist" }`)
		return
	}

	ctx.SetBodyString(`{ "success": "true" }`)
}
