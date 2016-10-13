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

func (m *Miogo) RemoveFile(path string) error {
	if file, ok := m.FetchFile(path); ok {

		var linksNumber map[string]int
		db.C("fs.files").Update(bson.M{"_id": file.FileID}, bson.M{"$inc": bson.M{"links": -1}})
		db.C("fs.files").Find(bson.M{"_id": file.FileID}).Select(bson.M{"links": 1}).One(&linksNumber)

		if linksNumber["links"] == 0 {
			if err := db.GridFS("fs").RemoveId(file.FileID); err != nil {
				log.Printf("RemoveId (GridFS) failed for FileID '%s' (%s)\n", file.FileID.String(), path)
				return errors.New("Error when removing file")
			}
		}

		d, f := formatF(path)
		if err := db.C("folders").Update(bson.M{"path": d}, bson.M{"$pull": bson.M{"files": bson.M{"name": f}}}); err != nil {
			return errors.New("Error when removing file")
		}

		m.filesCache.Invalidate(path)
		m.filesContentCache.Invalidate(path)
		m.foldersCache.Invalidate(d)

		return nil
	} else {
		return errors.New("File does not exist")
	}

	return nil
}

// TODO better handle for duplicate files
func (m *Miogo) CopyFile(path, dest, destFilename string, u *User) error {
	dest = formatD(dest)
	var parentFolderPath = parentD(dest)
	if parentFolder, ok := m.FetchFolder(parentFolderPath); ok {
		if GetRightType(u, parentFolder.Rights) < AllowedToWrite {
			return errors.New("Access denied")
		}
	} else {
		return errors.New("Destination folder does not exist")
	}
	var ok bool
	var sourceFile *File
	if sourceFile, ok = m.FetchFile(path); ok {
		if GetRightType(u, sourceFile.Rights) < AllowedToRead {
			return errors.New("Access denied")
		}
	} else {
		return errors.New("Source file does not exist")
	}

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
		return nil
	}
	return errors.New("Error when copying file")
}
