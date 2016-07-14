package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/mgo.v2/bson"
	"io"
	"net/http"
	"strings"
)

func (m *Miogo) GetFile(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	r.ParseForm()
	path := strings.TrimSpace(r.Form["path"][0])

	if folder, ok := m.db.GetFolderWithFile(path); ok {
		err := m.db.GetFile(w, folder.Files[0].FileID)

		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{ "error": "Server error" }`)
		}

		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{ "error": "File not found" }`)
}

func (m *Miogo) GetFolder(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	r.ParseForm()
	path := strings.TrimRight(strings.TrimSpace(r.Form["path"][0]), "/")
	w.Header().Set("Content-Type", "application/json")

	if len(path) == 0 {
		path = "/"
	}

	if folder, ok := m.db.GetFolder(path); ok {
		res, _ := json.Marshal(folder)
		fmt.Fprint(w, string(res))
		return
	}

	fmt.Fprint(w, `{ "error": "Folder does not exist" }`)
}

func (m *Miogo) NewFolder(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	r.ParseForm()
	path := strings.TrimRight(strings.TrimSpace(r.Form["path"][0]), "/")

	w.Header().Set("Content-Type", "application/json")

	pos := strings.LastIndex(path, "/")

	if pos > -1 {
		if _, ok := m.db.GetFolder(path[:pos+1]); ok {
			fmt.Fprintf(w, `{ "success": "%t" }`, m.db.NewFolder(path))
			return
		}
	}

	fmt.Fprint(w, `{ "error": "Bad folder name" }`)
}

func (m *Miogo) Upload(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	reader, err := r.MultipartReader()

	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		fmt.Fprint(w, `{ "error": "Bad request" }`)
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
				fmt.Fprint(w, `{ "error": "Failure on our side" }`)
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
		fmt.Fprint(w, `{ "error": "Wrong path" }`)
		return
	}

	fmt.Fprint(w, `{ "success": "true" }`)
}

<<<<<<< Updated upstream
func (m *Miogo) SetResourceRights(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	r.ParseForm()
	resource := strings.TrimSpace(r.Form["resource"][0])
	entityType := strings.TrimSpace(r.Form["type"][0])
	rights := strings.TrimSpace(r.Form["rights"][0])
	var err error
	if entityType == "user" || entityType == "group" {
		name := strings.TrimSpace(r.Form["name"][0])
		err = m.db.SetResourceRights(entityType, rights, resource, name)
	} else {
		err = m.db.SetResourceRights(entityType, rights, resource, "")
	}
	if err != nil {
		fmt.Fprint(w, `{"error" : "Can't set rights"}`)
		return
	}
	fmt.Fprint(w, `{"success" : "true"}`)
}
