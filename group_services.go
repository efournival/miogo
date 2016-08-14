package main

import (
	"strings"

	"github.com/valyala/fasthttp"
	"gopkg.in/mgo.v2/bson"
)

type Group struct {
	Id     string `bson:"_id,omitempty" json:"id"`
	Admins []User `json:"admins,omitempty"`
}

func (m *Miogo) FetchGroup(name string) (*Group, bool) {
	if val, ok := m.groupsCache.Get(name); ok {
		return val.(*Group), ok
	}

	query := db.C("groups").Find(bson.M{"_id": name})

	if count, err := query.Count(); count > 0 && err == nil {
		var group Group
		query.One(&group)

		m.groupsCache.Set(name, &group)

		return &group, true
	}

	return nil, false
}

func (m *Miogo) NewGroup(ctx *fasthttp.RequestCtx, u *User) {
	name := strings.TrimSpace(string(ctx.FormValue("name")))

	if _, exists := m.FetchGroup(name); exists {
		ctx.SetBodyString(`{ "error": "Group already exists" }`)
		return
	}

	db.C("groups").Insert(bson.M{"_id": name})
	ctx.SetBodyString(`{ "success": "true" }`)
}

func (m *Miogo) RemoveGroup(ctx *fasthttp.RequestCtx, u *User) {
	name := strings.TrimSpace(string(ctx.FormValue("name")))

	if _, exists := m.FetchGroup(name); !exists {
		ctx.SetBodyString(`{ "error": "Group does not exist" }`)
		return
	}

	// Store users belonging to the group
	var users []User
	db.C("users").Find(bson.M{"groups": bson.M{"$in": name}}).All(&users)

	db.C("users").Update(bson.M{"groups": bson.M{"$in": name}}, bson.M{"$pull": bson.M{"groups": name}})

	// Invalidate users by email
	var ukeys []string
	for _, user := range users {
		ukeys = append(ukeys, user.Email)
	}
	m.usersCache.Invalidate(ukeys...)

	db.C("groups").RemoveId(name)
	m.groupsCache.Invalidate(name)

	ctx.SetBodyString(`{ "success": "true" }`)
}

func (m *Miogo) AddUserToGroup(ctx *fasthttp.RequestCtx, u *User) {
	// TODO: handle multiple users
	user := strings.TrimSpace(string(ctx.FormValue("user")))
	group := strings.TrimSpace(string(ctx.FormValue("group")))

	//TODO: check if group and user exist

	db.C("users").Update(bson.M{"email": user}, bson.M{"$addToSet": bson.M{"groups": group}})

	m.usersCache.Invalidate(user)

	ctx.SetBodyString(`{ "success": "true" }`)
}

func (m *Miogo) RemoveUserFromGroup(ctx *fasthttp.RequestCtx, u *User) {
	user := strings.TrimSpace(string(ctx.FormValue("user")))
	group := strings.TrimSpace(string(ctx.FormValue("group")))

	//TODO: check if group and user exist

	db.C("users").Update(bson.M{"email": user}, bson.M{"$pull": bson.M{"groups": group}})

	m.usersCache.Invalidate(user)

	ctx.SetBodyString(`{ "success": "true" }`)
}

func (m *Miogo) SetGroupAdmin(ctx *fasthttp.RequestCtx, u *User) {
	user := strings.TrimSpace(string(ctx.FormValue("user")))
	group := strings.TrimSpace(string(ctx.FormValue("group")))

	//TODO: check if group and user exist

	db.C("groups").Update(bson.M{"_id": group}, bson.M{"$addToSet": bson.M{"admins": user}})
	ctx.SetBodyString(`{ "success": "true" }`)
}
