package main

import (
	"net/http"
	"net/url"
	"reflect"
	"runtime"
	"strings"
)

type ServiceOption int

const (
	NoJSON ServiceOption = (1 << iota)
	NoFormParsing
	NoLoginCheck
)

type ServiceFunc func(http.ResponseWriter, *http.Request, *User)

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
		pos := strings.LastIndex(s.Name, ".")
		s.Name = strings.Split(s.Name[pos+1:], "-")[0]
	}

	m.mux.HandleFunc("/"+s.Name, func(w http.ResponseWriter, r *http.Request) {
		var ok bool

		if r.Method != "POST" {
			http.Error(w, "Please send POST requests", http.StatusBadRequest)
			return
		}

		if s.Options^NoFormParsing > 0 {
			r.ParseForm()

			if ok = s.Validate(r.Form); !ok {
				http.Error(w, "Wrong arguments", http.StatusBadRequest)
				return
			}
		}

		var u *User

		if s.Options^NoLoginCheck > 0 {
			if u, ok = m.GetUserFromRequest(r); !ok {
				http.Error(w, "Not logged in", http.StatusForbidden)
				return
			}
		}

		if s.Options^NoJSON > 0 {
			w.Header().Set("Content-Type", "application/json")
		}

		s.Handler(w, r, u)
	})
}

func (s *Service) Validate(f url.Values) bool {
	for _, v := range s.MandatoryFields {
		if _, ok := f[v]; !ok {
			return false
		}
	}

	if len(s.AtLeastOneField) > 0 {
		good := false

		for _, v := range s.AtLeastOneField {
			if _, ok := f[v]; !ok {
				good = true
			}
		}

		return good
	}

	return true
}
