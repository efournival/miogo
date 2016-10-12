package main

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"

	"gopkg.in/mgo.v2/bson"
)

type File struct {
	Name   string        `bson:"name" json:"name"`
	FileID bson.ObjectId `bson:"file_id" json:"-"`
	Rights *Right        `bson:"rights,omitempty" json:"rights,omitempty"`
}

type FilesBulk struct {
	Files map[bson.ObjectId]string
	Path  string
}

func NewFilesBulk(path string) *FilesBulk {
	return &FilesBulk{Files: make(map[bson.ObjectId]string), Path: path}
}

func (fb *FilesBulk) AddFile(id bson.ObjectId, filename string) {
	fb.Files[id] = filename
}

func (fb *FilesBulk) Revert() {
	for id, _ := range fb.Files {
		db.GridFS("fs").RemoveId(id)
	}
}

func (m *Miogo) PushFilesBulk(fb *FilesBulk) {
	bulk := db.C("folders").Bulk()
	bulk.Unordered()

	for id, filename := range fb.Files {
		bulk.Update(bson.M{"path": fb.Path}, bson.M{"$push": bson.M{"files": bson.M{"name": filename, "file_id": id}}})
	}

	bulk.Run()
	m.foldersCache.Invalidate(fb.Path)
}

func (m *Miogo) CreateGFSFile(name string, file multipart.File) (bson.ObjectId, error) {
	gf, err := db.GridFS("fs").Create(name)

	if err != nil {
		log.Printf("Cannot create a GridFS file: %s\n", err)
		return bson.NewObjectId(), err
	}

	_, err = io.Copy(gf, file)
	gf.Close()

	gfId := gf.Id().(bson.ObjectId)
	db.C("fs.files").Update(bson.M{"_id": gfId}, bson.M{"$set": bson.M{"links": 1}})

	if err != nil {
		log.Printf("Cannot copy to GridFS: %s\n", err)
	}

	return gfId, err
}

func (m *Miogo) FetchFile(path string) (*File, bool) {
	path = formatD(path)

	if val, ok := m.filesCache.Get(path); ok {
		return val.(*File), true
	}

	d, f := formatF(path)
	query := db.C("folders").Find(bson.M{"path": d, "files.name": f}).
		Select(bson.M{"files": bson.M{"$elemMatch": bson.M{"name": f}}})

	if count, err := query.Count(); count > 0 && err == nil {
		var folder Folder
		query.One(&folder)

		m.filesCache.Set(path, &folder.Files[0])

		return &folder.Files[0], true
	}

	return nil, false
}

// TODO Rights, and a better handle for duplicate files
func (m *Miogo) CopyFile(path, dest, destFilename string) bool {
	dest = formatD(dest)
	sourceFile, _ := m.FetchFile(path)
	gfId := sourceFile.FileID
	if path == dest && destFilename == sourceFile.Name {
		destFilename = destFilename + "(DUPLICATE)"
	}
	if _, ok := m.FetchFile(dest + "/" + destFilename); ok {
		destFilename = destFilename + "(DUPLICATE)"
	}
	err := db.C("folders").Update(bson.M{"path": dest}, bson.M{"$push": bson.M{"files": bson.M{"name": destFilename, "file_id": gfId}}})
	if err == nil {
		db.C("fs.files").Update(bson.M{"_id": gfId}, bson.M{"$inc": bson.M{"links": 1}})
		return true
	}
	return false
}

func (m *Miogo) CopyFolder(path, dest string) bool {
	dest = formatD(dest)
	sourceFolder, _ := m.FetchFolder(path)

	if _, ok := m.FetchFolder(parentD(dest)); ok {
		if _, exists := m.FetchFolder(dest); !exists {
			m.foldersCache.Invalidate(parentD(dest))
			db.C("folders").Insert(bson.M{"path": dest})
		}
	} else {
		return false
	}
	for _, file := range sourceFolder.Files {
		m.CopyFile(sourceFolder.Path+"/"+file.Name, dest, file.Name)
	}
	for _, subFolder := range sourceFolder.Folders {
		db.C("folders").Insert(bson.M{"path": dest + subFolder.Path})
		if !m.CopyFolder(subFolder.Path, dest) {
			return false
		}
	}
	m.foldersCache.Invalidate(dest)
	return true
}

func (m *Miogo) FetchFileContent(path string, destination io.Writer, user *User) error {
	if file, ok := m.FetchFile(path); ok {
		if GetRightType(user, file.Rights) < AllowedToRead {
			return errors.New("Access denied")
		}

		if val, ok := m.filesContentCache.Get(path); ok {
			_, err := destination.Write(val.([]byte))
			return err
		}

		gfsfile, err := db.GridFS("fs").OpenId(file.FileID)

		if err != nil {
			log.Printf("Cannot get file from GridFS (%s): %s\n", file.FileID.String(), err)
			return err
		}

		defer gfsfile.Close()

		// If the file is too big, use a buffer
		if gfsfile.Size() < 64<<20 {
			b, err := ioutil.ReadAll(gfsfile)

			if err != nil {
				log.Printf("Cannot read from GridFS: %s\n", err)
				return err
			}

			m.filesContentCache.Set(path, b)

			_, err = destination.Write(b)
		} else {
			_, err = io.Copy(destination, gfsfile)
		}

		if err != nil {
			log.Printf("Cannot output file content: %s\n", err)
		}

		return err
	}

	return errors.New("File not found")
}

func (m *Miogo) RemoveFile(path string) bool {
	path = formatD(path)

	if file, ok := m.FetchFile(path); ok {

		var linksNumber map[string]int
		db.C("fs.files").Update(bson.M{"_id": file.FileID}, bson.M{"$inc": bson.M{"links": -1}})
		db.C("fs.files").Find(bson.M{"_id": file.FileID}).Select(bson.M{"links": 1}).One(&linksNumber)

		if linksNumber["links"] == 0 {
			if err := db.GridFS("fs").RemoveId(file.FileID); err != nil {
				log.Printf("RemoveId (GridFS) failed for FileID '%s' (%s)\n", file.FileID.String(), path)
				return false
			}
		}

		d, f := formatF(path)
		if err := db.C("folders").Update(bson.M{"path": d}, bson.M{"$pull": bson.M{"files": bson.M{"name": f}}}); err != nil {
			return false
		}

		m.filesCache.Invalidate(path)
		m.filesContentCache.Invalidate(path)
		m.foldersCache.Invalidate(d)

		return true
	}

	return false
}
