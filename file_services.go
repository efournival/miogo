package main

import (
	"bufio"
	"encoding/json"
	"strings"

	"github.com/valyala/fasthttp"
	"gopkg.in/mgo.v2/bson"
)

func (m *Miogo) GetFile(ctx *fasthttp.RequestCtx, u *User) {
	path := formatD(string(ctx.FormValue("path")))

	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		if err := m.FetchFileContent(path, w); err != nil {
			ctx.Response.Header.Add("Content-Type", "application/json")

			if err.Error() == "File not found" {
				w.WriteString(`{ "error": "File not found" }`)
			} else {
				w.WriteString(`{ "error": "Server error" }`)
			}
		}
	})
}

func (m *Miogo) GetFolder(ctx *fasthttp.RequestCtx, u *User) {
	path := formatD(string(ctx.FormValue("path")))

	if folder, ok := m.FetchFolder(path); ok {
		res, _ := json.Marshal(folder)
		ctx.SetBody(res)
		return
	}

	ctx.SetBodyString(`{ "error": "Folder does not exist" }`)
}

func (m *Miogo) NewFolder(ctx *fasthttp.RequestCtx, u *User) {
	path := formatD(string(ctx.FormValue("path")))

	if _, ok := m.FetchFolder(parentD(path)); !ok {
		ctx.SetBodyString(`{ "error": "Bad folder name" }`)
		return
	}

	if _, exists := m.FetchFolder(path); exists {
		ctx.SetBodyString(`{ "error": "Folder already exists" }`)
		return
	}

	m.foldersCache.Invalidate(parentD(path))

	db.C("folders").Insert(bson.M{"path": path})

	ctx.SetBodyString(`{ "success": "true" }`)
}

func (m *Miogo) Upload(ctx *fasthttp.RequestCtx, u *User) {
	form, err := ctx.MultipartForm()

	if err != nil {
		ctx.SetBodyString(`{ "error": "Bad request" }`)
		return
	}

	var path string

	if val, ok := form.Value["path"]; ok {
		path = strings.TrimSpace(val[0])

		if _, exists := m.FetchFolder(path); !exists {
			ctx.SetBodyString(`{ "error": "Wrong path" }`)
			return
		}
	}

	for _, header := range form.File["file"] {
		file, err := header.Open()

		if err != nil {
			ctx.SetBodyString(`{ "error": "Bad file header" }`)
			return
		}

		id, err := m.CreateGFSFile(header.Filename, file)

		if err != nil {
			ctx.SetBodyString(`{ "error": "Failure on our side" }`)
			// TODO: remove from GridFS
			return
		}

		m.PushFile(path, header.Filename, id)
	}

	ctx.SetBodyString(`{ "success": "true" }`)
}
