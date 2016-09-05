package main

import (
	"bufio"
	"encoding/json"

	"github.com/valyala/fasthttp"
	"gopkg.in/mgo.v2/bson"
)

func (m *Miogo) GetFile(ctx *fasthttp.RequestCtx, u *User) {
	path := formatD(string(ctx.FormValue("path")))

	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		if err := m.FetchFileContent(path, w, u); err != nil {
			ctx.Response.Header.Add("Content-Type", "application/json")
			w.WriteString(`{ "error": "` + err.Error() + `" }`)
		}
	})
}

func (m *Miogo) GetFolder(ctx *fasthttp.RequestCtx, u *User) {
	path := formatD(string(ctx.FormValue("path")))

	if folder, ok := m.FetchFolder(path); ok {
		if GetRightType(u, folder.Rights) < AllowedToRead {
			ctx.SetBodyString(`{ "error": "Access denied" }`)
			return
		}

		res, _ := json.Marshal(folder)
		ctx.SetBody(res)

		return
	}

	ctx.SetBodyString(`{ "error": "Folder does not exist" }`)
}

func (m *Miogo) NewFolder(ctx *fasthttp.RequestCtx, u *User) {
	path := formatD(string(ctx.FormValue("path")))

	if folder, ok := m.FetchFolder(parentD(path)); ok {
		if GetRightType(u, folder.Rights) < AllowedToRead {
			ctx.SetBodyString(`{ "error": "Access denied" }`)
			return
		}
	} else {
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

	path := formatD(form.Value["path"][0])

	if folder, ok := m.FetchFolder(path); ok {
		if GetRightType(u, folder.Rights) < AllowedToWrite {
			ctx.SetBodyString(`{ "error": "Access denied" }`)
			return
		}
	} else {
		ctx.SetBodyString(`{ "error": "Wrong path" }`)
		return
	}

	fb := NewFilesBulk(path)

	for _, header := range form.File["file"] {
		file, err := header.Open()

		if err != nil {
			ctx.SetBodyString(`{ "error": "Bad file header" }`)
			return
		}

		id, err := m.CreateGFSFile(header.Filename, file)

		if err != nil {
			fb.Revert()
			ctx.SetBodyString(`{ "error": "Failure on our side" }`)
			return
		}

		fb.AddFile(id, header.Filename)
	}

	m.PushFilesBulk(fb)
	ctx.SetBodyString(`{ "success": "true" }`)
}
