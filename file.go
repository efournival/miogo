package main

import (
	"io"
	"log"
	"mime/multipart"

	"gopkg.in/mgo.v2/bson"
)

type File struct {
	Name   string        `bson:"name" json:"name"`
	FileID bson.ObjectId `bson:"file_id" json:"-"`
	Rights *Right        `bson:"rights,omitempty" json:"rights,omitempty"`
}

// TODO: bulk files push
func (m *Miogo) PushFile(path, filename string, id bson.ObjectId) bool {
	m.foldersCache.Invalidate(path)
	return db.C("folders").Update(bson.M{"path": path}, bson.M{"$push": bson.M{"files": bson.M{"name": filename, "file_id": id}}}) == nil
}

func (m *Miogo) CreateGFSFile(part *multipart.Part) (bson.ObjectId, string, error) {
	file, err := db.GridFS("fs").Create(part.FileName())

	if err != nil {
		log.Printf("Cannot create a GridFS file: %s\n", err)
		return bson.NewObjectId(), "", err
	}

	defer file.Close()

	_, err = io.Copy(file, part)

	if err != nil {
		log.Printf("Cannot copy to GridFS: %s\n", err)
	}

	return file.Id().(bson.ObjectId), part.FileName(), err
}

func (m *Miogo) GetGFSFile(destination io.Writer, id bson.ObjectId) error {
	file, err := db.GridFS("fs").OpenId(id)

	if err != nil {
		log.Printf("Cannot get file from GridFS (%s): %s\n", id.String(), err)
		return err
	}

	defer file.Close()

	_, err = io.Copy(destination, file)

	if err != nil {
		log.Printf("Cannot copy from GridFS: %s\n", err)
	}

	return err
}
