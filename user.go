package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
)

type User struct {
	Mail     string  `bson:"mail" json:"mail"`
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

func (m *Miogo) getSessionUser(r *http.Request) (*User, bool) {
	ck, err := r.Cookie("session")

	if err != nil {
		return nil, false
	}

	val, _ := hex.DecodeString(ck.Value)
	hash := hash(val)

	if usr, ok := m.db.GetUserFromSession(hash); ok {
		return &usr, true
	}

	return nil, false
}

func (m *Miogo) loginOK(email, password string) bool {
	if usr, ok := m.db.GetUser(email); ok {
		return bcrypt.CompareHashAndPassword([]byte(usr.Password), []byte(password)) == nil
	}

	return false
}

func (m *Miogo) newUserSession(email string, w http.ResponseWriter) {
	randBytes := make([]byte, 16)
	_, err := rand.Read(randBytes)

	if err != nil {
		log.Panicf("Cannot read crypto secure bytes: %s\n", err)
	}

	m.db.SetUserSession(email, hash(randBytes))

	cookie := http.Cookie{Name: "session", Value: hex.EncodeToString(randBytes)}
	http.SetCookie(w, &cookie)
}
