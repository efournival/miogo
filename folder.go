package main

import "gopkg.in/mgo.v2/bson"

type Folder struct {
	Path    string   `bson:"path" json:"path"`
	Files   []File   `bson:"files" json:"files,omitempty"`
	Folders []Folder `json:"folders,omitempty"`
	Rights  *Right   `bson:"rights,omitempty" json:"rights,omitempty"`
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

// TODO: check rights
func (m *Miogo) RemoveFolder(path string) bool {
	path = formatD(path)
	var folder *Folder
	var ok bool
	if val, okcache := m.foldersCache.Get(path); okcache {
		folder = val.(*Folder)
		ok = true
	} else {
		folder, ok = m.FetchFolder(path)
	}
	if !ok {
		return false
	}
	for _, file := range folder.Files {
		m.RemoveFile(folder.Path + file.Name)
		m.filesCache.Invalidate(folder.Path + file.Name)
		m.filesContentCache.Invalidate(folder.Path + file.Name)
		m.foldersCache.Invalidate(folder.Path)
	}

	for _, subFolder := range folder.Folders {
		if !m.RemoveFolder(subFolder.Path) {
			return false
		}
		m.foldersCache.Invalidate(subFolder.Path)
	}

	if err := db.C("folders").Remove(bson.M{"path": path}); err != nil {
		return false
	}

	return true
}
