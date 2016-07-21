package main

import (
	"net/http"
	"strings"

	"gopkg.in/mgo.v2/bson"
)

type Group struct {
	Id     string `bson:"_id,omitempty" json:"id"`
	Admins []User `json:"admins,omitempty"`
}

func (m *Miogo) NewGroup(w http.ResponseWriter, r *http.Request, u *User) {
	name := strings.TrimSpace(r.Form["name"][0])

	// TODO: cache groups

	if count, err := db.C("groups").Find(bson.M{"_id": name}).Count(); count == 0 && err == nil {
		db.C("groups").Insert(bson.M{"_id": name})
		w.Write([]byte(`{ "success": "true" }`))
		return
	}

	w.Write([]byte(`{ "error": "Group already exists" }`))
}

func (m *Miogo) RemoveGroup(w http.ResponseWriter, r *http.Request, u *User) {
	name := strings.TrimSpace(r.Form["name"][0])

	// TODO: not working when user belongs to more than one group

	if count, err := db.C("groups").Find(bson.M{"_id": name}).Count(); count > 0 && err == nil {
		db.C("users").UpdateAll(bson.M{"groups": name}, bson.M{"$pull": bson.M{"groups": name}})
		// TODO: invalidate these...
		db.C("groups").RemoveId(name)
		w.Write([]byte(`{ "success": "true" }`))
		return
	}

	w.Write([]byte(`{ "error": "Group does not exist" }`))
}

func (m *Miogo) AddUserToGroup(w http.ResponseWriter, r *http.Request, u *User) {
	user := strings.TrimSpace(r.Form["user"][0])
	group := strings.TrimSpace(r.Form["group"][0])

	//TODO: check if group and user exist

	db.C("users").Update(bson.M{"email": user}, bson.M{"$addToSet": bson.M{"groups": group}})

	m.usersCache.Invalidate(user)

	w.Write([]byte(`{ "success": "true" }`))
}

func (m *Miogo) RemoveUserFromGroup(w http.ResponseWriter, r *http.Request, u *User) {
	user := strings.TrimSpace(r.Form["user"][0])
	group := strings.TrimSpace(r.Form["group"][0])

	//TODO: check if group and user exist

	db.C("users").Update(bson.M{"email": user}, bson.M{"$pull": bson.M{"groups": group}})

	m.usersCache.Invalidate(user)

	w.Write([]byte(`{ "success": "true" }`))
}

func (m *Miogo) SetGroupAdmin(w http.ResponseWriter, r *http.Request, u *User) {
	user := strings.TrimSpace(r.Form["user"][0])
	group := strings.TrimSpace(r.Form["group"][0])

	//TODO: check if group and user exist

	db.C("groups").Update(bson.M{"_id": group}, bson.M{"$addToSet": bson.M{"admins": user}})
	w.Write([]byte(`{ "success": "true" }`))
}
