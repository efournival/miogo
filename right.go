package main

import (
	"net/http"
	"strings"

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

func (m *Miogo) SetResourceRights(w http.ResponseWriter, r *http.Request, u *User) {
	rights := strings.TrimSpace(r.Form["rights"][0])
	resource := formatD(r.Form["resource"][0])
	var (
		name, entityType string
	)

	if _, ok := r.Form["user"]; ok {
		name = strings.TrimSpace(r.Form["user"][0])
		entityType = "users"
	} else if _, ok := r.Form["group"]; ok {
		name = strings.TrimSpace(r.Form["group"][0])
		entityType = "groups"
	} else if _, ok := r.Form["all"]; ok {
		entityType = "all"
	}

	var err error

	if _, ok := m.FetchFolder(resource); ok {
		selector := bson.M{"path": bson.M{"$regex": bson.RegEx{`^` + resource, ""}}}

		// TODO: set child files rights?

		if entityType == "all" {
			_, err = db.C("folders").UpdateAll(selector, bson.M{"$set": bson.M{"rights.all": rights}})
		} else {
			_, err = db.C("folders").UpdateAll(selector, bson.M{"$addToSet": bson.M{"rights." + entityType: bson.M{"name": name, "rights": rights}}})
		}

		if err != nil {
			w.Write([]byte(`{ "error": "Cannot set rights" }`))
			return
		}

		m.foldersCache.InvalidateStartWith(resource)
	} else if _, ok := m.FetchFolderWithFile(resource); ok {
		dir, file := formatF(resource)

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
			w.Write([]byte(`{ "error": "Cannot set rights" }`))
			return
		}

		m.foldersCache.Invalidate(dir)
	} else {
		w.Write([]byte(`{ "error": "Resource does not exist" }`))
		return
	}

	w.Write([]byte(`{ "success": "true" }`))
}
