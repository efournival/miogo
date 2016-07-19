package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"gopkg.in/mgo.v2/bson"
)

func (m *Miogo) GetFile(w http.ResponseWriter, r *http.Request, u *User) {
	path := strings.TrimSpace(r.Form["path"][0])

	if folder, ok := m.db.GetFolderWithFile(path); ok {
		err := m.db.GetFile(w, folder.Files[0].FileID)

		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{ "error": "Server error" }`))
		}

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{ "error": "File not found" }`))
}

func (m *Miogo) GetFolder(w http.ResponseWriter, r *http.Request, u *User) {
	path := strings.TrimRight(strings.TrimSpace(r.Form["path"][0]), "/")

	if len(path) == 0 {
		path = "/"
	}

	if folder, ok := m.db.GetFolder(path); ok {
		res, _ := json.Marshal(folder)
		fmt.Fprint(w, string(res))
		return
	}

	w.Write([]byte(`{ "error": "Folder does not exist" }`))
}

func (m *Miogo) NewFolder(w http.ResponseWriter, r *http.Request, u *User) {
	path := strings.TrimRight(strings.TrimSpace(r.Form["path"][0]), "/")

	if _, ok := m.db.GetFolder(path[:strings.LastIndex(path, "/")]); ok {
		fmt.Fprintf(w, `{ "success": "%t" }`, m.db.NewFolder(path))
		return
	}

	w.Write([]byte(`{ "error": "Bad folder name" }`))
}

func (m *Miogo) Upload(w http.ResponseWriter, r *http.Request, u *User) {
	reader, err := r.MultipartReader()

	if err != nil {
		w.Write([]byte(`{ "error": "Bad request" }`))
		return
	}

	files := make(map[bson.ObjectId]string)
	path := "/"

	for {
		part, err := reader.NextPart()

		if err == io.EOF {
			break
		}

		if part.FormName() == "path" {
			buf := new(bytes.Buffer)
			buf.ReadFrom(part)
			path = buf.String()
		} else if part.FormName() == "file" {
			id, name, err := m.db.CreateFile(part)

			if err != nil {
				w.Write([]byte(`{ "error": "Failure on our side" }`))
				// TODO: remove from GridFS
				return
			}

			files[id] = name
		}
	}

	if _, ok := m.db.GetFolder(path); ok {
		for id, name := range files {
			m.db.PushFile(path, name, id)
		}
	} else {
		// TODO: remove from GridFS
		w.Write([]byte(`{ "error": "Wrong path" }`))
		return
	}

	w.Write([]byte(`{ "success": "true" }`))
}

func (m *Miogo) SetResourceRights(w http.ResponseWriter, r *http.Request, u *User) {
	var err error

	rights := strings.TrimSpace(r.Form["rights"][0])
	resource := strings.TrimSpace(r.Form["resource"][0])

	if _, ok := r.Form["user"]; ok {
		username := strings.TrimSpace(r.Form["user"][0])
		err = m.db.SetResourceRights("users", rights, resource, username)
	} else if _, ok := r.Form["group"]; ok {
		groupname := strings.TrimSpace(r.Form["group"][0])
		err = m.db.SetResourceRights("groups", rights, resource, groupname)
	} else {
		err = m.db.SetResourceRights("all", rights, resource, "")
	}

	if err != nil {
		w.Write([]byte(`{ "error": "Can't set rights" }`))
		return
	}

	w.Write([]byte(`{ "success" : "true" }`))
}
