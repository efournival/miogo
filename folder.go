package main

import (
	"errors"
	"log"

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
	if val, okcache := m.foldersCache.Get(path); okcache {
		folder = val.(*Folder)
		ok = true
	} else {
		folder, ok = m.FetchFolder(path)
	}
	m.foldersCache.Invalidate(path)
	if !ok {
		return errors.New("Folder to remove doesn't exist")
	}
	if GetRightType(u, folder.Rights) < AllowedToWrite {
		return errors.New("Access denied")
	}
	for _, file := range folder.Files {
		m.filesCache.Invalidate(folder.Path + "/" + file.Name)
		m.filesContentCache.Invalidate(folder.Path + "/" + file.Name)
		m.foldersCache.Invalidate(folder.Path)

		fileErr := m.RemoveFile(folder.Path+"/"+file.Name, u)
		if fileErr != nil {
			return fileErr
		}
	}

	for _, subFolder := range folder.Folders {

		err := m.RemoveFolder(subFolder.Path, u)
		if err != nil {
			return errors.New("Can't remove folder")
		}
	}

	if err := db.C("folders").Remove(bson.M{"path": path}); err != nil {
		return errors.New("Can't remove folder")
	}
	return nil
}

func (m *Miogo) CopyFolder(path, dest, destFoldername string, u *User) error {
	dest = formatD(dest)

	var destinationFolder = dest + destFoldername
	if dest != "/" {
		destinationFolder = dest + "/" + destFoldername
	}
	log.Println(destinationFolder)
	var parentFolderPath = parentD(dest)

	if parentFolder, ok := m.FetchFolder(parentFolderPath); ok {
		if GetRightType(u, parentFolder.Rights) < AllowedToWrite {
			return errors.New("Access denied")
		}
	} else {
		return errors.New("parent folder does not exist")
	}

	var ok bool
	var sourceFolder *Folder

	if sourceFolder, ok = m.FetchFolder(path); ok {
		if GetRightType(u, sourceFolder.Rights) < AllowedToRead {
			return errors.New("Access denied")
		}
	} else {
		return errors.New("Source folder does not exist")
	}

	if _, ok := m.FetchFolder(dest); ok {
		if _, exists := m.FetchFolder(destinationFolder); !exists {
			m.foldersCache.Invalidate(dest)
			// if it does not, create our destination folder
			db.C("folders").Insert(bson.M{"path": destinationFolder})
		}
	} else {
		return errors.New("Error when copying folder")
	}

	for _, file := range sourceFolder.Files {
		m.CopyFile(sourceFolder.Path+"/"+file.Name, destinationFolder, file.Name, u)
	}

	for _, subFolder := range sourceFolder.Folders {
		m.foldersCache.Invalidate(dest)
		m.foldersCache.Invalidate(subFolder.Path)

		_, folderName := formatF(subFolder.Path)
		db.C("folders").Insert(bson.M{"path": destinationFolder + "/" + folderName})
		err := m.CopyFolder(subFolder.Path, destinationFolder, folderName, u)
		if err != nil {
			return err
		}
	}
	return nil
}
