package main

import (
	"errors"

	"gopkg.in/mgo.v2/bson"
)

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

func (m *Miogo) RemoveFolder(path string, u *User) error {
	var folder *Folder
	var ok bool

	if folder, ok = m.FetchFolder(path); !ok {
		return errors.New("Folder to remove doesn't exist")
	}

	if GetRightType(u, folder.Rights) < AllowedToWrite {
		return errors.New("Access denied")
	}

	for _, file := range folder.Files {
		if GetRightType(u, file.Rights) < AllowedToWrite {
			return errors.New("Access denied")
		}

		if err := m.RemoveFile(folder.Path + "/" + file.Name); err != nil {
			return err
		}

		m.filesCache.Invalidate(folder.Path + "/" + file.Name)
		m.filesContentCache.Invalidate(folder.Path + "/" + file.Name)
		m.foldersCache.Invalidate(folder.Path)
	}

	for _, subFolder := range folder.Folders {
		err := m.RemoveFolder(subFolder.Path, u)

		if err != nil {
			return errors.New("Cannot remove folder")
		}

		m.foldersCache.Invalidate(folder.Path)
	}

	if err := db.C("folders").Remove(bson.M{"path": path}); err != nil {
		return errors.New("Cannot remove folder")
	}

	m.foldersCache.Invalidate(path)

	return nil
}

func (m *Miogo) CopyFolder(path, dest, destFoldername string, u *User) error {
	dest = formatD(dest)

	var destinationFolder string

	if dest != "/" {
		destinationFolder = dest + "/" + destFoldername
	} else {
		destinationFolder = dest + destFoldername
	}

	var sourceFolder Folder

	if sourceFolder, ok := m.FetchFolder(path); ok {
		if GetRightType(u, sourceFolder.Rights) < AllowedToRead {
			return errors.New("Access denied")
		}
	} else {
		return errors.New("Source folder does not exist")
	}

	if _, ok := m.FetchFolder(dest); ok {
		if _, exists := m.FetchFolder(destinationFolder); !exists {
			db.C("folders").Insert(bson.M{"path": destinationFolder})
		}
	} else {
		return errors.New("Destination folder does not exist")
	}

	for _, file := range sourceFolder.Files {
		if GetRightType(u, file.Rights) < AllowedToRead {
			return errors.New("Access denied")
		}
		m.CopyFile(sourceFolder.Path+"/"+file.Name, destinationFolder, file.Name, u)
	}

	for _, subFolder := range sourceFolder.Folders {
		_, folderName := formatF(subFolder.Path)
		db.C("folders").Insert(bson.M{"path": destinationFolder + "/" + folderName})

		if err := m.CopyFolder(subFolder.Path, destinationFolder, folderName, u); err != nil {
			return err
		}

		m.foldersCache.Invalidate(subFolder.Path)
	}

	m.foldersCache.Invalidate(dest)

	return nil
}
