package main

import (
	"errors"
	"strings"

	"github.com/valyala/fasthttp"

	"gopkg.in/mgo.v2/bson"
)

type RightType int

const (
	Nothing RightType = iota
	AllowedToRead
	AllowedToWrite
	AllowedToChangeRights
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

func RightStringToRightType(str string) RightType {
	switch str {
	case "n":
		return Nothing
	case "r":
		return AllowedToRead
	case "rw":
		return AllowedToWrite
	case "rwa":
		return AllowedToChangeRights
	}

	return Nothing
}

func UserBelongsToGroup(u *User, g string) bool {
	for _, group := range u.Groups {
		if group == g {
			return true
		}
	}

	return false
}

func GetRightType(u *User, r *Right) RightType {
	if r == nil {
		// WARNING: this is the default policy when there is NO rights set
		return AllowedToWrite
	}

	result := RightStringToRightType(r.All)

	if result == AllowedToChangeRights {
		// Stop here as we are returning the most permissive right
		return result
	}

	for _, er := range r.Users {
		// TODO: by user ID instead
		if er.Name == u.Email {
			rights := RightStringToRightType(er.Rights)

			if rights > result {
				if rights == AllowedToChangeRights {
					return rights
				}
				result = rights
			}
		}
	}

	for _, er := range r.Groups {
		// TODO: by group ID instead
		if UserBelongsToGroup(u, er.Name) {
			rights := RightStringToRightType(er.Rights)
			if rights > result {
				result = rights
			}
		}
	}

	return result
}

func (m *Miogo) SetResourceRightsP(resource, rights, entityType, name string) error {
	var err error

	if folder, ok := m.FetchFolder(resource); ok {
		// TODO: server admin should have 'rwa' access to everything
		// TODO: fix and uncomment the following code
		/*if GetRightType(u, folder.Rights) < AllowedToChangeRights {
			ctx.SetBodyString(`{ "error": "Access denied" }`)
			return
		}*/

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
	} else if _, ok := m.FetchFile(resource); ok {
		/*if GetRightType(u, f.Rights) < AllowedToChangeRights {
			ctx.SetBodyString(`{ "error": "Access denied" }`)
			return
		}*/

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
			return errors.New("Can't set rights for file")
		}

		m.filesCache.Invalidate(resource)
	}

	return nil
}

func (m *Miogo) SetResourceRights(ctx *fasthttp.RequestCtx, u *User) error {
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

	if _, ok := m.FetchFolder(resource); ok {
		if err := m.SetResourceRightsP(resource, rights, entityType, name); err != nil {
			return err
		}

		m.foldersCache.InvalidateStartWith(resource)
	} else if _, ok := m.FetchFile(resource); ok {
		if err := m.SetResourceRightsP(resource, rights, entityType, name); err == nil {
			return errors.New("Cannot set file rights")
		}

		dir, _ := formatF(resource)
		m.foldersCache.Invalidate(dir)
		m.filesCache.Invalidate(resource)
	} else {
		return errors.New("Resource does not exist")
	}

	ctx.SetBodyString(jsonkv("success", "true"))
	return nil
}
