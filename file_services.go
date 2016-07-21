package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"gopkg.in/mgo.v2/bson"
)

func (m *Miogo) GetFile(w http.ResponseWriter, r *http.Request, u *User) {
	path := formatD(r.Form["path"][0])

	if folder, ok := m.FetchFolderWithFile(path); ok {
		err := m.GetGFSFile(w, folder.Files[0].FileID)

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
	path := formatD(r.Form["path"][0])

	if folder, ok := m.FetchFolder(path); ok {
		res, _ := json.Marshal(folder)
		fmt.Fprint(w, string(res))
		return
	}

	w.Write([]byte(`{ "error": "Folder does not exist" }`))
}

func (m *Miogo) NewFolder(w http.ResponseWriter, r *http.Request, u *User) {
	path := formatD(r.Form["path"][0])

	if _, ok := m.FetchFolder(parentD(path)); !ok {
		w.Write([]byte(`{ "error": "Bad folder name" }`))
		return
	}

	if _, exists := m.FetchFolder(path); exists {
		w.Write([]byte(`{ "error": "Folder already exists" }`))
		return
	}

	m.foldersCache.Invalidate(parentD(path))

	db.C("folders").Insert(bson.M{"path": path})

	w.Write([]byte(`{ "success": "true" }`))
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
			path = formatD(buf.String())
		} else if part.FormName() == "file" {
			id, name, err := m.CreateGFSFile(part)

			if err != nil {
				w.Write([]byte(`{ "error": "Failure on our side" }`))
				// TODO: remove from GridFS
				return
			}

			files[id] = name
		}
	}

	if _, ok := m.FetchFolder(path); ok {
		for id, name := range files {
			m.PushFile(path, name, id)
		}
	} else {
		// TODO: remove from GridFS
		w.Write([]byte(`{ "error": "Wrong path" }`))
		return
	}

	w.Write([]byte(`{ "success": "true" }`))
}
