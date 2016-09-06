package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/valyala/fasthttp"

	"gopkg.in/mgo.v2/bson"

	"golang.org/x/crypto/bcrypt"
)

/*
 * Login:
 *   1. User's password is checked
 *   2. Session is created in DB
 *   3. Raw session is cached (unsecure but only in RAM)
 *   4. Cookie is set
 *
 * Access to a service requiring login:
 *   1. Cookie is fetched
 *   2. If raw session is in cache, go to 4
 *   3. Otherwise, hash raw session and check if an user matches
 *   4. Update session expiration then return user to the service
 */

type User struct {
	Email    string   `bson:"email" json:"email"`
	Password string   `bson:"password" json:"password"`
	Groups   []string `bson:"groups" json:"groups,omitempty"`
	Session  struct {
		Hash       string `bson:"hash"`
		Expiration int64  `bson:"expire"`
	} `bson:"session,omitempty" json:"-"`
}

func hash(val []byte) string {
	hasher := sha256.New()
	hasher.Write(val)
	return hex.EncodeToString(hasher.Sum(nil))
}

func (m *Miogo) newUserSession(usr *User, ctx *fasthttp.RequestCtx) {
	randBytes := make([]byte, 16)
	_, err := rand.Read(randBytes)

	if err != nil {
		log.Panicf("Cannot read crypto secure bytes: %s\n", err)
	}

	raw := hex.EncodeToString(randBytes)

	usr.Session.Hash = hash(randBytes)
	usr.Session.Expiration = bson.Now().Add(m.sessionDuration).Unix()

	db.C("users").Update(bson.M{"email": usr.Email}, usr)

	m.sessionsCache.Set(raw, usr)

	// The cookie will have a "session" duration on the client side (until the browser is closed)
	cookie := fasthttp.AcquireCookie()
	cookie.SetHTTPOnly(true)
	cookie.SetKey("session")
	cookie.SetValue(raw)
	ctx.Response.Header.SetCookie(cookie)
	fasthttp.ReleaseCookie(cookie)
}

func (m *Miogo) Login(ctx *fasthttp.RequestCtx, u *User) error {
	if usr, ok := m.FetchUser(strings.TrimSpace(string(ctx.FormValue("email")))); ok {
		if err := bcrypt.CompareHashAndPassword([]byte(usr.Password), []byte(ctx.FormValue("password"))); err == nil {
			m.newUserSession(usr, ctx)
			ctx.SetBodyString(jsonkv("success", "true"))
			return nil
		}

		return errors.New("Wrong password")
	}

	return errors.New("User does not exist")
}

func (m *Miogo) Logout(ctx *fasthttp.RequestCtx, u *User) error {
	raw := string(ctx.Request.Header.Cookie("session"))

	if raw == u.Session.Hash {
		m.sessionsCache.Invalidate(raw)
		db.C("users").Update(bson.M{"session.hash": u.Session.Hash}, bson.M{"$unset": "session"})
	}

	ctx.Response.Header.DelClientCookie("session")

	ctx.SetBodyString(jsonkv("success", "true"))
	return nil
}

func (m *Miogo) NewUser(ctx *fasthttp.RequestCtx, u *User) error {
	email := string(ctx.FormValue("email"))
	hashedPassword, _ := bcrypt.GenerateFromPassword(ctx.FormValue("password"), bcrypt.DefaultCost)

	if _, exists := m.FetchUser(email); exists {
		return errors.New("User already exists")
	}

	db.C("users").Insert(bson.M{"email": email, "password": string(hashedPassword)})

	ctx.SetBodyString(jsonkv("success", "true"))
	return nil
}

func (m *Miogo) RemoveUser(ctx *fasthttp.RequestCtx, u *User) error {
	email := string(ctx.FormValue("email"))

	if _, exists := m.FetchUser(email); !exists {
		return errors.New("User does not exist")
	}

	db.C("users").Remove(bson.M{"email": email})
	m.usersCache.Invalidate(email)

	ctx.SetBodyString(jsonkv("success", "true"))
	return nil
}

func (m *Miogo) updateUserSession(usr *User, raw string) (*User, bool) {
	if usr.Session.Expiration < time.Now().Unix() {
		return nil, false
	}

	usr.Session.Expiration = time.Now().Add(m.sessionDuration).Unix()

	// If time until session expiration is enough, don't annoy MongoDB
	if time.Unix(usr.Session.Expiration, 0).Sub(time.Now()) < m.sessionDuration/4 {
		db.C("users").Update(bson.M{"session.hash": usr.Session.Hash}, usr)
	}

	m.sessionsCache.Set(raw, usr)

	return usr, true
}

func (m *Miogo) GetUserFromRequest(ctx *fasthttp.RequestCtx) (*User, bool) {
	raw := string(ctx.Request.Header.Cookie("session"))

	if val, ok := m.sessionsCache.Get(raw); ok {
		return m.updateUserSession(val.(*User), raw)
	}

	val, _ := hex.DecodeString(raw)
	query := db.C("users").Find(bson.M{"session.hash": hash(val)})

	if count, err := query.Count(); count > 0 && err == nil {
		var user User
		query.One(&user)
		return m.updateUserSession(&user, raw)
	}

	return nil, false
}

func (m *Miogo) FetchUser(email string) (*User, bool) {
	if val, ok := m.usersCache.Get(email); ok {
		return val.(*User), ok
	}

	query := db.C("users").Find(bson.M{"email": email})

	if count, err := query.Count(); count > 0 && err == nil {
		var user User
		query.One(&user)

		m.usersCache.Set(email, &user)

		return &user, true
	}

	return nil, false
}
