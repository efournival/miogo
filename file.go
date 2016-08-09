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

	defer gf.Close()

	_, err = io.Copy(gf, file)

	if err != nil {
		log.Printf("Cannot copy to GridFS: %s\n", err)
	}

	return gf.Id().(bson.ObjectId), err
}

func (m *Miogo) FetchFileContent(path string, destination io.Writer) error {
	if val, ok := m.filesCache.Get(path); ok {
		_, err := destination.Write(val.([]byte))
		return err
	}

	d, f := formatF(path)
	query := db.C("folders").Find(bson.M{"path": d, "files.name": f}).
		Select(bson.M{"files": bson.M{"$elemMatch": bson.M{"name": f}}})

	if count, err := query.Count(); count > 0 && err == nil {
		var folder Folder
		query.One(&folder)

		file, err := db.GridFS("fs").OpenId(folder.Files[0].FileID)

		if err != nil {
			log.Printf("Cannot get file from GridFS (%s): %s\n", folder.Files[0].FileID.String(), err)
			return err
		}

		defer file.Close()

		// If the file is too big, use a buffer
		if file.Size() < 64<<20 {
			b, err := ioutil.ReadAll(file)

			if err != nil {
				log.Printf("Cannot read from GridFS: %s\n", err)
				return err
			}

			m.filesCache.Set(path, b)

			_, err = destination.Write(b)
		} else {
			_, err = io.Copy(destination, file)
		}

		if err != nil {
			log.Printf("Cannot output file content: %s\n", err)
		}

		return err
	}

	return errors.New("File not found")
}

func (m *Miogo) FileExists(path string) bool {
	if _, ok := m.filesCache.Get(path); ok {
		return true
	}

	d, f := formatF(path)
	query := db.C("folders").Find(bson.M{"path": d, "files.name": f}).
		Select(bson.M{"files": bson.M{"$elemMatch": bson.M{"name": f}}})

	if count, err := query.Count(); count > 0 && err == nil {
		return true
	}

	return false
}

func (m *Miogo) RemoveFile(path string) bool {
	d, f := formatF(path)

	query := db.C("folders").Find(bson.M{"path": d, "files.name": f}).
		Select(bson.M{"files": bson.M{"$elemMatch": bson.M{"name": f}}})

	if count, err := query.Count(); count > 0 && err == nil {
		var folder Folder
		query.One(&folder)

		if err := db.GridFS("fs").RemoveId(folder.Files[0].FileID); err != nil {
			return false
		}

		if err := db.C("folders").Update(bson.M{"path": d}, bson.M{"$pull": bson.M{"files": bson.M{"name": f}}}); err != nil {
			return false
		}

		return true
	}

	return false
}
