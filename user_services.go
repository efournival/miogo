package main

import (
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func (m *Miogo) Login(w http.ResponseWriter, r *http.Request, u *User) {
	email := strings.TrimSpace(r.Form["email"][0])
	password := strings.TrimSpace(r.Form["password"][0])

	if m.loginOK(email, password) {
		m.newUserSession(email, w)
		w.Write([]byte(`{ "success": "true" }`))
		return
	}

	w.Write([]byte(`{ "success": "false" }`))
}

func (m *Miogo) NewUser(w http.ResponseWriter, r *http.Request, u *User) {
	mail := strings.TrimSpace(r.Form["email"][0])
	password := []byte(strings.TrimSpace(r.Form["password"][0]))
	hashedPassword, _ := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	err := m.db.NewUser(mail, string(hashedPassword))
	if err != nil {
		w.Write([]byte(`{ "error": "` + err.Error() + `" }`))
		return
	}
	w.Write([]byte(`{ "success": "true" }`))
}

func (m *Miogo) RemoveUser(w http.ResponseWriter, r *http.Request, u *User) {
	mail := strings.TrimSpace(r.Form["email"][0])
	err := m.db.RemoveUser(mail)
	if err != nil {
		w.Write([]byte(`{ "error": "Cannot remove user" }`))
		return
	}
	w.Write([]byte(`{ "success": "true" }`))
}

func (m *Miogo) NewGroup(w http.ResponseWriter, r *http.Request, u *User) {
	group := strings.TrimSpace(r.Form["group"][0])
	err := m.db.NewGroup(group)
	if err != nil {
		w.Write([]byte(`{ "error": "Cannot create group" }`))
		return
	}
	w.Write([]byte(`{ "success": "true" }`))
}

func (m *Miogo) RemoveGroup(w http.ResponseWriter, r *http.Request, u *User) {
	group := strings.TrimSpace(r.Form["group"][0])
	err := m.db.RemoveGroup(group)
	if err != nil {
		w.Write([]byte(`{ "error": "Cannot remove group" }`))
		return
	}
	w.Write([]byte(`{ "success": "true" }`))
}

func (m *Miogo) AddUserToGroup(w http.ResponseWriter, r *http.Request, u *User) {
	userMail := strings.TrimSpace(r.Form["user"][0])
	group := strings.TrimSpace(r.Form["group"][0])
	err := m.db.AddUserToGroup(userMail, group)
	if err != nil {
		w.Write([]byte(`{ "error": "Cannot add user to group" }`))
		return
	}
	w.Write([]byte(`{ "success": "true" }`))
}

func (m *Miogo) RemoveUserFromGroup(w http.ResponseWriter, r *http.Request, u *User) {
	userMail := strings.TrimSpace(r.Form["user"][0])
	group := strings.TrimSpace(r.Form["group"][0])
	err := m.db.RemoveUserFromGroup(userMail, group)
	if err != nil {
		w.Write([]byte(`{ "error": "Cannot remove user from group" }`))
		return
	}
	w.Write([]byte(`{ "success": "true" }`))
}

func (m *Miogo) SetGroupAdmin(w http.ResponseWriter, r *http.Request, u *User) {
	admin := strings.TrimSpace(r.Form["user"][0])
	group := strings.TrimSpace(r.Form["group"][0])
	err := m.db.SetGroupAdmin(admin, group)
	if err != nil {
		w.Write([]byte(`{ "error": "Cannot set admin for group" }`))
		return
	}
	w.Write([]byte(`{ "success": "true" }`))
}
