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

type CacheEntry struct {
	// TODO: limit caching with time (first, in the configuration file; then implement a more efficient system)
	Expire time.Time
	Value  interface{}
}

type MiogoDB struct {
	db           *mgo.Database
	delay        time.Duration
	filesCache   map[string]CacheEntry
	foldersCache map[string]CacheEntry
}

func NewMiogoDB(host string, cacheTime int) *MiogoDB {
	session, err := mgo.Dial(host)
	if err != nil {
		log.Fatalf("Cannot connect to MongoDB: %s", err)
	}

	selector := bson.M{"path": "/"}

	// Init DB if it's the first time Miogo is launched
	if count, err := session.DB("miogo").C("folders").Find(selector).Count(); count == 0 && err == nil {
		session.DB("miogo").C("folders").Insert(selector)
	}

	return &MiogoDB{
		session.DB("miogo"),
		time.Duration(cacheTime) * time.Minute,
		make(map[string]CacheEntry),
		make(map[string]CacheEntry),
	}
}

func (mdb *MiogoDB) GetFolder(path string) (Folder, bool) {
	if val, ok := mdb.foldersCache[path]; ok {
		return val.Value.(Folder), true
	}

	query := mdb.db.C("folders").Find(bson.M{"path": path})

	if count, err := query.Count(); count > 0 && err == nil {
		var folder Folder
		query.One(&folder)

		var subfolders []Folder
		mdb.db.C("folders").Find(bson.M{"path": bson.RegEx{"^" + strings.TrimRight(path, "/") + "/*[^/]+$", ""}}).Select(bson.M{"path": 1}).All(&subfolders)
		folder.Folders = append(folder.Folders, subfolders...)

		mdb.foldersCache[path] = CacheEntry{time.Now().Add(mdb.delay), folder}

		return folder, true
	}

	return Folder{}, false
}

func (mdb *MiogoDB) GetFolderWithFile(path string) (Folder, bool) {
	if val, ok := mdb.filesCache[path]; ok {
		return val.Value.(Folder), true
	}

	pos := strings.LastIndex(path, "/")
	name := path[pos+1:]

	query := mdb.db.C("folders").Find(bson.M{"path": path[:pos+1], "files.name": name}).Select(bson.M{"files": bson.M{"$elemMatch": bson.M{"name": name}}})

	if count, err := query.Count(); count > 0 && err == nil {
		var folder Folder
		query.One(&folder)
		mdb.filesCache[path] = CacheEntry{time.Now().Add(mdb.delay), folder}
		return folder, true
	}

	return Folder{}, false
}

// TODO: bulk files push
func (mdb *MiogoDB) PushFile(path, filename string, id bson.ObjectId) bool {
	mdb.invalidateFolder(path)
	return mdb.db.C("folders").Update(bson.M{"path": path}, bson.M{"$push": bson.M{"files": bson.M{"name": filename, "file_id": id}}}) == nil
}

func (mdb *MiogoDB) NewFolder(path string) bool {
	parent := path[:strings.LastIndex(path, "/")+1]

	if len(parent) == 0 {
		parent = "/"
	}

	mdb.invalidateFolder(parent)

	return mdb.db.C("folders").Insert(bson.M{"path": path}) == nil
}

func (mdb *MiogoDB) CreateFile(part *multipart.Part) (bson.ObjectId, string, error) {
	file, err := mdb.db.GridFS("fs").Create(part.FileName())
	defer file.Close()

	if err != nil {
		log.Println("Cannot create a GridFS file: %s", err)
		return bson.NewObjectId(), "", err
	}

	_, err = io.Copy(file, part)

	if err != nil {
		log.Println("Cannot copy to GridFS: %s", err)
	}

	return file.Id().(bson.ObjectId), part.FileName(), err
}

func (mdb *MiogoDB) GetFile(destination io.Writer, id bson.ObjectId) error {
	file, err := mdb.db.GridFS("fs").OpenId(id)
	defer file.Close()

	if err != nil {
		log.Println("Cannot get file from GridFS (%s): %s", id.String(), err)
		return err
	}

	_, err = io.Copy(destination, file)

	if err != nil {
		log.Println("Cannot copy from GridFS: %s", err)
	}

	return err
}

func (mdb *MiogoDB) invalidateFolder(path string) {
	delete(mdb.foldersCache, path)
}

func (mdb *MiogoDB) NewUser(mail string, password string) error {
	return mdb.db.C("users").Insert(bson.M{"mail": mail, "password": password})
}

func (mdb *MiogoDB) NewGroup(name string) error {
	return mdb.db.C("groups").Insert(bson.M{"_id": name})
}

func (mdb *MiogoDB) AddUserToGroup(user string, group string) error {
    // get id with mail from db
    userId bson.ObjectId
    mdb.db.C("users").Find({"mail" : user}).Select(bson.M{"_id" :1).One(&userId)
    // insert nested attribute group with a set
}
