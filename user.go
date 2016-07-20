package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
)

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

func (m *Miogo) GetSessionUser(r *http.Request) (*User, bool) {
	ck, err := r.Cookie("session")

	if err != nil {
		return nil, false
	}

	if usr, ok := m.db.GetUserFromSession(ck.Value); ok {
		return &usr, true
	}

	return nil, false
}

func (m *Miogo) NewUserSession(usr User, w http.ResponseWriter) {
	randBytes := make([]byte, 16)
	_, err := rand.Read(randBytes)
	raw := hex.EncodeToString(randBytes)

	if err != nil {
		log.Panicf("Cannot read crypto secure bytes: %s\n", err)
	}

	m.db.SetUserSession(usr, raw, hash(randBytes))
	http.SetCookie(w, &http.Cookie{Name: "session", Value: raw})
}
