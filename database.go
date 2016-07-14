package main

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"log"
	"mime/multipart"
	"strings"
	"time"
)

type MiogoDB struct {
	db           *mgo.Database
	filesCache   *Cache
	foldersCache *Cache
}

func NewMiogoDB(host string, cacheTime int) *MiogoDB {
	session, err := mgo.Dial(host)
	if err != nil {
		log.Panicf("Cannot connect to MongoDB: %s\n", err)
	}

	selector := bson.M{"path": "/"}

	// Init DB if it's the first time Miogo is launched
	if count, err := session.DB("miogo").C("folders").Find(selector).Count(); count == 0 && err == nil {
		session.DB("miogo").C("folders").Insert(selector)
	}

	dur := time.Duration(cacheTime) * time.Minute

	return &MiogoDB{
		session.DB("miogo"),
		NewCache(dur),
		NewCache(dur),
	}
}

func (mdb *MiogoDB) GetFolder(path string) (Folder, bool) {
	if val, ok := mdb.foldersCache.Get(path); ok {
		return val.(Folder), true
	}

	query := mdb.db.C("folders").Find(bson.M{"path": path})

	if count, err := query.Count(); count > 0 && err == nil {
		var folder Folder
		query.One(&folder)

		var subfolders []Folder
		mdb.db.C("folders").Find(bson.M{"path": bson.RegEx{"^" + strings.TrimRight(path, "/") + "/*[^/]+$", ""}}).Select(bson.M{"path": 1}).All(&subfolders)
		folder.Folders = append(folder.Folders, subfolders...)

		mdb.foldersCache.Set(path, folder)

		return folder, true
	}

	return Folder{}, false
}

func (mdb *MiogoDB) GetFolderWithFile(path string) (Folder, bool) {
	if val, ok := mdb.filesCache.Get(path); ok {
		return val.(Folder), true
	}

	pos := strings.LastIndex(path, "/")
	name := path[pos+1:]
	folderPath := path[:pos]

	if len(folderPath) == 0 {
		folderPath = "/"
	}

	query := mdb.db.C("folders").Find(bson.M{"path": folderPath, "files.name": name}).Select(bson.M{"files": bson.M{"$elemMatch": bson.M{"name": name}}})

	if count, err := query.Count(); count > 0 && err == nil {
		var folder Folder
		query.One(&folder)
		mdb.filesCache.Set(path, folder)
		return folder, true
	}

	return Folder{}, false
}

// TODO: bulk files push
func (mdb *MiogoDB) PushFile(path, filename string, id bson.ObjectId) bool {
	mdb.foldersCache.Invalidate(path)
	return mdb.db.C("folders").Update(bson.M{"path": path}, bson.M{"$push": bson.M{"files": bson.M{"name": filename, "file_id": id}}}) == nil
}

func (mdb *MiogoDB) NewFolder(path string) bool {
	parent := path[:strings.LastIndex(path, "/")+1]

	if len(parent) == 0 {
		parent = "/"
	}

	mdb.foldersCache.Invalidate(parent)

	return mdb.db.C("folders").Insert(bson.M{"path": path}) == nil
}

func (mdb *MiogoDB) CreateFile(part *multipart.Part) (bson.ObjectId, string, error) {
	file, err := mdb.db.GridFS("fs").Create(part.FileName())
	defer file.Close()

	if err != nil {
		log.Printf("Cannot create a GridFS file: %s\n", err)
		return bson.NewObjectId(), "", err
	}

	_, err = io.Copy(file, part)

	if err != nil {
		log.Printf("Cannot copy to GridFS: %s\n", err)
	}

	return file.Id().(bson.ObjectId), part.FileName(), err
}

func (mdb *MiogoDB) GetFile(destination io.Writer, id bson.ObjectId) error {
	file, err := mdb.db.GridFS("fs").OpenId(id)
	defer file.Close()

	if err != nil {
		log.Printf("Cannot get file from GridFS (%s): %s\n", id.String(), err)
		return err
	}

	_, err = io.Copy(destination, file)

	if err != nil {
		log.Printf("Cannot copy from GridFS: %s\n", err)
	}

	return err
}

func (mdb *MiogoDB) NewUser(mail string, password string) error {
	return mdb.db.C("users").Insert(bson.M{"mail": mail, "password": password})
}

func (mdb *MiogoDB) RemoveUser(mail string) error {
	return mdb.db.C("users").Remove(bson.M{"mail": mail})
}

func (mdb *MiogoDB) NewGroup(name string) error {
	return mdb.db.C("groups").Insert(bson.M{"_id": name})
}

func (mdb *MiogoDB) RemoveGroup(name string) error {
	_, err := mdb.db.C("users").UpdateAll(bson.M{"groups": name}, bson.M{"$pull": bson.M{"groups": name}})
	if err == nil {
		err = mdb.db.C("groups").RemoveId(name)
	}
	return err
}

func (mdb *MiogoDB) AddUserToGroup(user string, group string) error {
	//TODO : check if group and user exists beforehand?
	return mdb.db.C("users").Update(bson.M{"mail": user}, bson.M{"$addToSet": bson.M{"groups": group}})
}

func (mdb *MiogoDB) RemoveUserFromGroup(user string, group string) error {
	//TODO : check if group and user exists beforehand?
	return mdb.db.C("users").Update(bson.M{"mail": user}, bson.M{"$pull": bson.M{"groups": group}})
}

func (mdb *MiogoDB) SetGroupAdmin(user string, group string) error {
	return mdb.db.C("groups").Update(bson.M{"_id": group}, bson.M{"$addToSet": bson.M{"admins": user}})
}

func (mdb *MiogoDB) SetResourceRights(entityType string, rights string, resource string, name string) error {
	if name == "" {
		name = "all"
	}
	if count, err := mdb.db.C("folders").Find(bson.M{"path": resource}).Count(); count == 0 && err == nil {
		pos := strings.LastIndex(resource, "/")
		filename := resource[pos+1:]
		path := resource[:pos]
		if name == "all" {
			return mdb.db.C("folders").Update(bson.M{"path": path, "files.name": filename}, bson.M{"$addToSet": bson.M{"files.0.rights": bson.M{"all": "rw"}}})
		} else {
			return mdb.db.C("folders").Update(bson.M{"path": path, "files.name": filename}, bson.M{"$addToSet": bson.M{"files.0.rights." + entityType: bson.M{"name": name, "rights": rights}}})
		}
	} else {
		if name == "all" {
			return mdb.db.C("folders").Update(bson.M{"path": resource}, bson.M{"$addToSet": bson.M{"rights": bson.M{"all": "rw"}}})
		} else {
			return mdb.db.C("folders").Update(bson.M{"path": resource}, bson.M{"$addToSet": bson.M{"rights." + entityType: bson.M{"name": name, "rights": rights}}})
		}
	}
}
