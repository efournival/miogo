package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"strings"
	"time"

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
	Email    string  `bson:"email" json:"email"`
	Password string  `bson:"password" json:"password"`
	Groups   []Group `json:"groups,omitempty"`
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

func (m *Miogo) newUserSession(usr *User, w http.ResponseWriter) {
	randBytes := make([]byte, 16)
	_, err := rand.Read(randBytes)

	if err != nil {
		log.Panicf("Cannot read crypto secure bytes: %s\n", err)
	}

	raw := hex.EncodeToString(randBytes)

	usr.Session.Hash = hash(randBytes)
	usr.Session.Expiration = bson.Now().Add(m.sessionDuration).Unix()

	db.C("users").Update(bson.M{"email": usr.Email}, bson.M{"$set": bson.M{"session": bson.M{"hash": usr.Session.Hash, "expiration": usr.Session.Expiration}}})

	m.sessionsCache.Set(raw, usr)

	// The cookie will have a "session" duration on the client side (until the browser is closed)
	http.SetCookie(w, &http.Cookie{Name: "session", Value: raw})
}

func (m *Miogo) Login(w http.ResponseWriter, r *http.Request, u *User) {
	email := strings.TrimSpace(r.Form["email"][0])
	password := strings.TrimSpace(r.Form["password"][0])

	if usr, ok := m.GetUser(email); ok {
		if err := bcrypt.CompareHashAndPassword([]byte(usr.Password), []byte(password)); err == nil {
			m.newUserSession(usr, w)
			w.Write([]byte(`{ "success": "true" }`))
			return
		}
	}

	w.Write([]byte(`{ "success": "false" }`))
}

func (m *Miogo) Logout(w http.ResponseWriter, r *http.Request, u *User) {
	ck, err := r.Cookie("session")

	if err != nil {
		m.sessionsCache.Invalidate(ck.Value)
	}

	db.C("users").Update(bson.M{"session.hash": u.Session.Hash}, bson.M{"$unset": "session"})

	http.SetCookie(w, &http.Cookie{Name: "session", Value: "", MaxAge: -1})

	w.Write([]byte(`{ "success": "true" }`))
}

func (m *Miogo) NewUser(w http.ResponseWriter, r *http.Request, u *User) {
	email := strings.TrimSpace(r.Form["email"][0])
	password := strings.TrimSpace(r.Form["password"][0])
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if _, exists := m.GetUser(email); exists {
		w.Write([]byte(`{ "error": "User already exists" }`))
		return
	}

	db.C("users").Insert(bson.M{"email": email, "password": string(hashedPassword)})

	w.Write([]byte(`{ "success": "true" }`))
}

func (m *Miogo) RemoveUser(w http.ResponseWriter, r *http.Request, u *User) {
	email := strings.TrimSpace(r.Form["email"][0])

	if _, exists := m.GetUser(email); !exists {
		w.Write([]byte(`{ "error": "User does not exist" }`))
		return
	}

	db.C("users").Remove(bson.M{"email": email})
	w.Write([]byte(`{ "success": "true" }`))
}

func (m *Miogo) updateUserSession(usr *User, raw string) (*User, bool) {
	if usr.Session.Expiration < time.Now().Unix() {
		return nil, false
	}

	usr.Session.Expiration = time.Now().Add(m.sessionDuration).Unix()
	db.C("users").Update(bson.M{"session.hash": usr.Session.Hash}, bson.M{"session.expiration": usr.Session.Expiration})
	m.sessionsCache.Set(raw, usr)

	return usr, true
}

func (m *Miogo) GetUserFromRequest(r *http.Request) (*User, bool) {
	ck, err := r.Cookie("session")

	if err != nil {
		return nil, false
	}

	raw := ck.Value

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

func (m *Miogo) GetUser(email string) (*User, bool) {
	// TODO: user cache

	query := db.C("users").Find(bson.M{"email": email})

	if count, err := query.Count(); count > 0 && err == nil {
		var user User
		query.One(&user)
		return &user, true
	}

	return nil, false
}
