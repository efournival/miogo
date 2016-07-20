package main

import (
	"errors"
	"io"
	"log"
	"mime/multipart"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type MiogoDB struct {
	db              *mgo.Database
	sessionDuration time.Duration
	filesCache      *Cache
	foldersCache    *Cache
	usersCache      *Cache
}

func NewMiogoDB(host string, cacheTime int, sessionDuration int, adminEmail string, adminPassword string) *MiogoDB {
	session, err := mgo.Dial(host)
	if err != nil {
		log.Panicf("Cannot connect to MongoDB: %s\n", err)
	}

	selector := bson.M{"path": "/"}

	// Init DB if it's the first time Miogo is launched
	if count, err := session.DB("miogo").C("folders").Find(selector).Count(); count == 0 && err == nil {
		session.DB("miogo").C("folders").Insert(selector)
	}

	if count, err := session.DB("miogo").C("users").Find(bson.M{"admin": "true"}).Count(); count == 0 && err == nil {
		hashedAdminPassword, _ := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
		session.DB("miogo").C("users").Insert(bson.M{"email": adminEmail, "password": string(hashedAdminPassword), "admin": "true"})
	}

	dur := time.Duration(cacheTime) * time.Minute

	return &MiogoDB{
		session.DB("miogo"),
		time.Duration(sessionDuration) * time.Minute,
		NewCache(dur),
		NewCache(dur),
		NewCache(dur),
	}
}

func (mdb *MiogoDB) updateUserSession(usr User) User {
	usr.Session.Expiration = time.Now().Add(mdb.sessionDuration).Unix()
	mdb.db.C("users").Update(bson.M{"session.hash": usr.Session.Hash}, bson.M{"session.expiration": usr.Session.Expiration})
	mdb.usersCache.Set(usr.Session.Hash, usr)

	return usr
}

func (mdb *MiogoDB) GetUser(email string) (User, bool) {
	query := mdb.db.C("users").Find(bson.M{"email": email})

	if count, err := query.Count(); count > 0 && err == nil {
		var user User
		query.One(&user)
		return user, true
	}

	return User{}, false
}

func (mdb *MiogoDB) GetUserFromSession(session string) (User, bool) {
	if val, ok := mdb.usersCache.Get(session); ok {
		if val.(User).Session.Expiration > time.Now().Unix() {
			return mdb.updateUserSession(val.(User)), true
		}

		return User{}, false
	}

	query := mdb.db.C("users").Find(bson.M{"session.hash": session})

	if count, err := query.Count(); count > 0 && err == nil {
		var user User
		query.One(&user)

		if user.Session.Expiration > time.Now().Unix() {
			return User{}, false
		}

		return mdb.updateUserSession(user), true
	}

	return User{}, false
}

func (mdb *MiogoDB) SetUserSession(email, hash string) {
	mdb.db.C("users").Update(bson.M{"email": email}, bson.M{"$set": bson.M{"session": bson.M{"hash": hash, "expiration": bson.Now().Add(mdb.sessionDuration).Unix()}}})
}

func (mdb *MiogoDB) RemoveUserSession(usr *User) {
	mdb.db.C("users").Update(bson.M{"session.hash": usr.Session.Hash}, bson.M{"$unset": "session"})
}

func (mdb *MiogoDB) GetFolder(path string) (Folder, bool) {
	if len(path) == 0 {
		path = "/"
	}

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

	if _, exists := mdb.GetFolder(path); exists {
		return false
	} else {
		return mdb.db.C("folders").Insert(bson.M{"path": path}) == nil
	}
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
	if _, exists := mdb.GetUser(mail); exists {
		return errors.New("user already exists")
	} else {
		return mdb.db.C("users").Insert(bson.M{"email": mail, "password": password})
	}
}

func (mdb *MiogoDB) RemoveUser(mail string) error {
	return mdb.db.C("users").Remove(bson.M{"email": mail})
}

func (mdb *MiogoDB) NewGroup(name string) error {
	if count, err := mdb.db.C("groups").Find(bson.M{"_id": name}).Count(); count != 0 && err == nil {
		return errors.New("group already exists")
	} else {
		return mdb.db.C("groups").Insert(bson.M{"_id": name})
	}
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
	return mdb.db.C("users").Update(bson.M{"email": user}, bson.M{"$addToSet": bson.M{"groups": group}})
}

func (mdb *MiogoDB) RemoveUserFromGroup(user string, group string) error {
	//TODO : check if group and user exists beforehand?
	return mdb.db.C("users").Update(bson.M{"email": user}, bson.M{"$pull": bson.M{"groups": group}})
}

func (mdb *MiogoDB) SetGroupAdmin(user string, group string) error {
	return mdb.db.C("groups").Update(bson.M{"_id": group}, bson.M{"$addToSet": bson.M{"admins": user}})
}

func (mdb *MiogoDB) SetResourceRights(entityType string, rights string, resource string, name string) error {
	resource = strings.TrimRight(resource, "/")

	// TODO: make a reusable function
	if len(resource) == 0 {
		resource = "/"
	}

	if _, ok := mdb.GetFolder(resource); ok {
		var err error
		//selector := bson.M{"path": bson.RegEx{`^` + resource, ""}}
		selector := bson.M{"path": bson.M{"$regex": bson.RegEx{`^` + resource, ""}}}

		// TODO: set child files rights?

		if entityType == "all" {
			_, err = mdb.db.C("folders").UpdateAll(selector, bson.M{"$set": bson.M{"rights.all": rights}})
		} else {
			_, err = mdb.db.C("folders").UpdateAll(selector, bson.M{"$addToSet": bson.M{"rights." + entityType: bson.M{"name": name, "rights": rights}}})
		}

		if err == nil {
			mdb.foldersCache.InvalidateStartWith(resource)
		}

		return err
	}

	if _, ok := mdb.GetFolderWithFile(resource); ok {
		pos := strings.LastIndex(resource, "/")
		filename := resource[pos+1:]
		path := resource[:pos]
		mdb.foldersCache.Invalidate(path)

		if len(path) == 0 {
			path = "/"
		}

		if entityType == "all" {
			return mdb.db.C("folders").Update(
				bson.M{"path": path, "files.name": filename},
				bson.M{"$set": bson.M{"files.0.rights.all": rights}})
		}

		return mdb.db.C("folders").Update(
			bson.M{"path": path, "files.name": filename},
			bson.M{"$addToSet": bson.M{"files.0.rights." + entityType: bson.M{"name": name, "rights": rights}}})
	}

	return errors.New("Resource does not exist")
}
