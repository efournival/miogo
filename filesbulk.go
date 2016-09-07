package main

import "gopkg.in/mgo.v2/bson"

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
