package main

import (
	"reflect"
	"runtime"
	"strings"

	"github.com/valyala/fasthttp"
)

type ServiceOption int

const (
	NoJSON ServiceOption = (1 << iota)
	NoLoginCheck
)

type ServiceFunc func(*fasthttp.RequestCtx, *User)

type Service struct {
	Name            string
	Handler         ServiceFunc
	Options         ServiceOption
	MandatoryFields []string
	AtLeastOneField []string
}

func (m *Miogo) RegisterService(s *Service) {
	if s.Name == "" {
		s.Name = runtime.FuncForPC(reflect.ValueOf(s.Handler).Pointer()).Name()
		s.Name = strings.Split(s.Name[strings.LastIndex(s.Name, ".")+1:], "-")[0]
	}

	m.services["/"+s.Name] = func(ctx *fasthttp.RequestCtx) {
		var ok bool

		if !ctx.Request.Header.IsPost() {
			ctx.Error("Please send POST requests", fasthttp.StatusBadRequest)
			return
		}

		if ok = s.Validate(ctx.PostArgs()); !ok {
			ctx.Error("Wrong arguments", fasthttp.StatusBadRequest)
			return
		}

		var u *User

		if s.Options&NoLoginCheck == 0 {
			if u, ok = m.GetUserFromRequest(ctx); !ok {
				ctx.Error("Not logged in", fasthttp.StatusForbidden)
				return
			}
		}

		if s.Options&NoJSON == 0 {
			ctx.Response.Header.Add("Content-Type", "application/json")
		}

		s.Handler(ctx, u)
	}
}

func (s *Service) Validate(a *fasthttp.Args) bool {
	for _, v := range s.MandatoryFields {
		if !a.Has(v) {
			return false
		}
	}

	if len(s.AtLeastOneField) > 0 {
		for _, v := range s.AtLeastOneField {
			if a.Has(v) {
				return true
			}
		}

		return false
	}

	return true
}
