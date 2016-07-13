package main

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strings"
)

func (m *Miogo) NewUser(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	r.ParseForm()
	mail := strings.TrimSpace(r.Form["mail"][0])
	password := []byte(strings.TrimSpace(r.Form["password"][0]))
	hashedPassword, _ := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	err := m.db.NewUser(mail, string(hashedPassword))
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		fmt.Fprint(w, `{"error" : "Can't create user"}`)
		return
	}
	fmt.Fprint(w, `{ "success": "true" }`)
}

func (m *Miogo) RemoveUser(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	r.ParseForm()
	mail := strings.TrimSpace(r.Form["mail"][0])
	err := m.db.RemoveUser(mail)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		fmt.Fprint(w, `{"error" : "Can't remove user"}`)
		return
	}
	fmt.Fprint(w, `{ "success": "true" }`)
}


func (m *Miogo) NewGroup(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	r.ParseForm()
	group := strings.TrimSpace(r.Form["group"][0])
	err := m.db.NewGroup(group)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		fmt.Fprint(w, `{"error" : "Can't create group"}`)
		return
	}
	fmt.Fprint(w, `{"success" : "true"}`)
}

func (m *Miogo) RemoveGroup(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	r.ParseForm()
	group := strings.TrimSpace(r.Form["group"][0])
	err := m.db.RemoveGroup(group)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		fmt.Fprint(w, `{"error" : "Can't remove group"}`)
		return
	}
	fmt.Fprint(w, `{"success" : "true"}`)
}

func (m *Miogo) AddUserToGroup(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	r.ParseForm()
	userMail := strings.TrimSpace(r.Form["user"][0])
	group := strings.TrimSpace(r.Form["group"][0])
	err := m.db.AddUserToGroup(userMail, group)
	if err != nil {
		fmt.Fprint(w, `{"error" : "Can't add user to group"}`)
		return
	}
	fmt.Fprint(w, `{"success" : "true"}`)
}

func (m *Miogo) RemoveUserFromGroup(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	r.ParseForm()
	userMail := strings.TrimSpace(r.Form["user"][0])
	group := strings.TrimSpace(r.Form["group"][0])
	err := m.db.RemoveUserFromGroup(userMail, group)
	if err != nil {
		fmt.Fprint(w, `{"error" : "Can't remove user from group"}`)
		return
	}
	fmt.Fprint(w, `{"success" : "true"}`)
}

func (m *Miogo) SetGroupAdmin(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	r.ParseForm()
	admin := strings.TrimSpace(r.Form["user"][0])
	group := strings.TrimSpace(r.Form["group"][0])
	err := m.db.SetGroupAdmin(admin, group)
	if err != nil {
		fmt.Fprint(w, `{"error" : "Can't set admin for group"}`)
		return
	}
	fmt.Fprint(w, `{"success" : "true"}`)
}
