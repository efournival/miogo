package main

import (
	"encoding/json"
	"errors"

	"github.com/valyala/fasthttp"
	"gopkg.in/mgo.v2/bson"
)

func (m *Miogo) GetFile(ctx *fasthttp.RequestCtx, u *User) error {
	path := formatD(string(ctx.FormValue("path")))
	return m.FetchFileContent(path, ctx.Response.BodyWriter(), u)
}

func (m *Miogo) GetFolder(ctx *fasthttp.RequestCtx, u *User) error {
	path := formatD(string(ctx.FormValue("path")))

	if folder, ok := m.FetchFolder(path); ok {
		if GetRightType(u, folder.Rights) < AllowedToRead {
			return errors.New("Access denied")
		}

		res, _ := json.Marshal(folder)
		ctx.SetBody(res)
		return nil
	}

	return errors.New("Folder does not exist")
}

func (m *Miogo) NewFolder(ctx *fasthttp.RequestCtx, u *User) error {
	path := formatD(string(ctx.FormValue("path")))

	if folder, ok := m.FetchFolder(parentD(path)); ok {
		if GetRightType(u, folder.Rights) < AllowedToWrite {
			return errors.New("Access denied")
		}
	} else {
		return errors.New("Bad folder name")
	}

	if _, exists := m.FetchFolder(path); exists {
		return errors.New("Folder already exists")
	}

	m.foldersCache.Invalidate(parentD(path))

	db.C("folders").Insert(bson.M{"path": path})

	ctx.SetBodyString(jsonkv("success", "true"))
	return nil
}

func (m *Miogo) Upload(ctx *fasthttp.RequestCtx, u *User) error {
	form, err := ctx.MultipartForm()

	if err != nil {
		return errors.New("Bad request")
	}

	path := formatD(form.Value["path"][0])

	if folder, ok := m.FetchFolder(path); ok {
		if GetRightType(u, folder.Rights) < AllowedToWrite {
			return errors.New("Access denied")
		}
	} else {
		return errors.New("Wrong path")
	}

	fb := NewFilesBulk(path)

	for _, header := range form.File["file"] {
		file, err := header.Open()

		if err != nil {
			return errors.New("Bad file header")
		}

		id, err := m.CreateGFSFile(header.Filename, file)

		if err != nil {
			fb.Revert()
			return errors.New("Failure on our side")
		}

		fb.AddFile(id, header.Filename)
	}

	m.PushFilesBulk(fb)

	ctx.SetBodyString(jsonkv("success", "true"))
	return nil
}
