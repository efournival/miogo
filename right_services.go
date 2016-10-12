package main

import (
	"errors"
	"strings"

	"gopkg.in/mgo.v2/bson"

	"github.com/valyala/fasthttp"
)

func (m *Miogo) setFileRights(path, d, f, rights, entityType, entityName string) error {
	if entityType == "all" {
		db.C("folders").Update(
			bson.M{"path": d, "files.name": f},
			bson.M{"$set": bson.M{"files.0.rights.all": rights}})
	} else {
		db.C("folders").Update(
			bson.M{"path": d, "files.name": f},
			bson.M{"$addToSet": bson.M{"files.0.rights." + entityType: bson.M{"name": entityName, "rights": rights}}})
	}

	m.filesCache.Invalidate(path)

	return nil
}

func (m *Miogo) setFolderRights(u *User, path, rights, entityType, entityName string) error {
	selector := bson.M{"path": bson.M{"$regex": bson.RegEx{`^` + path, ""}}}
	query := db.C("folders").Find(selector)

	var folders []Folder
	query.All(&folders)

	for _, folder := range folders {
		for _, file := range folder.Files {
			if GetRightType(u, file.Rights) < AllowedToChangeRights {
				return errors.New("Access denied")
			}

			fullPath := strings.TrimSuffix(folder.Path, "/") + "/" + file.Name
			d, f := formatF(fullPath)

			if err := m.setFileRights(fullPath, d, f, rights, entityType, entityName); err != nil {
				return err
			}
		}
	}

	if entityType == "all" {
		db.C("folders").UpdateAll(selector, bson.M{"$set": bson.M{"rights.all": rights}})
	} else {
		db.C("folders").Update(selector, bson.M{"$addToSet": bson.M{"rights." + entityType: bson.M{"name": entityName, "rights": rights}}})
	}

	return nil
}

func (m *Miogo) SetResourceRights(ctx *fasthttp.RequestCtx, u *User) error {
	rights := strings.TrimSpace(string(ctx.FormValue("rights")))
	resource := formatD(string(ctx.FormValue("resource")))

	var (
		entityName, entityType string
	)

	if len(ctx.FormValue("user")) > 0 {
		entityName = strings.TrimSpace(string(ctx.FormValue("user")))
		entityType = "users"
	} else if len(ctx.FormValue("group")) > 0 {
		entityName = strings.TrimSpace(string(ctx.FormValue("group")))
		entityType = "groups"
	} else {
		entityType = "all"
	}

	if folder, ok := m.FetchFolder(resource); ok {
		if GetRightType(u, folder.Rights) < AllowedToChangeRights {
			return errors.New("Access denied")
		}

		if err := m.setFolderRights(u, resource, rights, entityType, entityName); err != nil {
			return err
		}

		m.foldersCache.InvalidateStartWith(resource)
	} else if file, ok := m.FetchFile(resource); ok {
		if GetRightType(u, file.Rights) < AllowedToChangeRights {
			return errors.New("Access denied")
		}

		d, f := formatF(resource)

		if err := m.setFileRights(resource, d, f, rights, entityType, entityName); err != nil {
			return err
		}

		m.foldersCache.Invalidate(d)
	} else {
		return errors.New("Resource does not exist")
	}

	ctx.SetBodyString(jsonkv("success", "true"))
	return nil
}
