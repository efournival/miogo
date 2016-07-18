package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var (
	miogo  *Miogo
	server *httptest.Server
)

func init() {
	miogo = NewMiogo()
	server = httptest.NewServer(miogo.mux)
}

func testPOST(t *testing.T, service, params, expected string) (bool, string) {
	request, err := http.NewRequest("POST", server.URL+"/"+service, strings.NewReader(params))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := http.DefaultClient.Do(request)

	if err != nil {
		return false, err.Error()
	}

	if res.StatusCode != 200 {
		return false, fmt.Sprintf("Expected response code 200 but got %s", res.StatusCode)
	}

	b, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return false, err.Error()
	}

	if expected != "" {
		if !strings.EqualFold(expected, string(b)) {
			return false, fmt.Sprintf("Expected: '%s', got: '%s'", expected, string(b))
		}
	}

	return true, ""
}

func testSuccessfulPOST(t *testing.T, service, params, expected string) {
	if ok, err := testPOST(t, service, params, expected); !ok {
		t.Error(err)
	}
}

func testFailPOST(t *testing.T, service, params string) {
	if ok, _ := testPOST(t, service, params, ""); ok {
		t.Error("Test should have failed but succeeded")
	}
}

func TestLogin(t *testing.T) {
	testSuccessfulPOST(t, "Login", fmt.Sprintf("email=%sXXX&password=%s", miogo.conf.AdminEmail, miogo.conf.AdminPassword), `{ "success": "false" }`)
	testSuccessfulPOST(t, "Login", fmt.Sprintf("email=%s&password=%sXXX", miogo.conf.AdminEmail, miogo.conf.AdminPassword), `{ "success": "false" }`)
	testSuccessfulPOST(t, "Login", fmt.Sprintf("email=%s&password=%s", miogo.conf.AdminEmail, miogo.conf.AdminPassword), `{ "success": "true" }`)
}
