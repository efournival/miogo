package main

import (
	"strings"

	"gopkg.in/mgo.v2/bson"
)

type Folder struct {
	Path    string   `bson:"path" json:"path"`
	Files   []File   `bson:"files" json:"files,omitempty"`
	Folders []Folder `json:"folders,omitempty"`
	Rights  *Right   `bson:"rights,omitempty" json:"rights,omitempty"`
}

func formatD(res string) string {
	res = strings.TrimRight(strings.TrimSpace(res), "/")

	if len(res) == 0 {
		return "/"
	}

	return res
}

func parentD(res string) string {
	return formatD(res[:strings.LastIndex(res, "/")+1])
}

func formatF(res string) (dir string, file string) {
	pos := strings.LastIndex(res, "/")
	file = res[pos+1:]
	dir = formatD(res[:pos])
	return
}

func (m *Miogo) FetchFolder(path string) (*Folder, bool) {
	if val, ok := m.foldersCache.Get(path); ok {
		return val.(*Folder), ok
	}

	query := db.C("folders").Find(bson.M{"path": path})

	if count, err := query.Count(); count > 0 && err == nil {
		var folder Folder
		query.One(&folder)

		var subfolders []Folder
		db.C("folders").Find(bson.M{"path": bson.RegEx{"^" + path + "/*[^/]+$", ""}}).Select(bson.M{"path": 1}).All(&subfolders)
		folder.Folders = append(folder.Folders, subfolders...)

		m.foldersCache.Set(path, &folder)

		return &folder, true
	}

	return nil, false
}

func (m *Miogo) FetchFolderWithFile(path string) (*Folder, bool) {
	if val, ok := m.filesCache.Get(path); ok {
		return val.(*Folder), ok
	}

	dir, file := formatF(path)

	query := db.C("folders").Find(bson.M{"path": dir, "files.name": file}).Select(bson.M{"files": bson.M{"$elemMatch": bson.M{"name": file}}})

	if count, err := query.Count(); count > 0 && err == nil {
		var folder Folder
		query.One(&folder)

		m.filesCache.Set(path, &folder)

		return &folder, true
	}

	return nil, false
}
