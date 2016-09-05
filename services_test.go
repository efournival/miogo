package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
)

var (
	miogo   *Miogo
	session string
)

func init() {
	miogo = NewMiogo()
	server := &fasthttp.Server{Handler: miogo.GetHandler()}
	go server.ListenAndServe(":8080")
}

func downloadAndHash(path string) string {
	request, err := http.NewRequest("POST", "http://localhost:8080/GetFile", strings.NewReader("path="+path))

	if err != nil {
		return ""
	}

	request.AddCookie(&http.Cookie{Name: "session", Value: session})
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := http.DefaultClient.Do(request)

	if err != nil {
		return ""
	}

	defer res.Body.Close()

	hash := md5.New()
	io.Copy(hash, res.Body)

	return fmt.Sprintf("%x", hash.Sum(nil))
}

func hashFile(file string) string {
	f, err := os.Open(file)

	if err != nil {
		return ""
	}

	defer f.Close()

	hash := md5.New()

	if _, err := io.Copy(hash, f); err != nil {
		return ""
	}

	return fmt.Sprintf("%x", hash.Sum(nil))
}

func upload(file, path, expected string) (bool, string) {
	f, err := os.Open(file)

	if err != nil {
		return false, err.Error()
	}

	defer f.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(file))

	if err != nil {
		return false, err.Error()
	}

	_, err = io.Copy(part, f)

	if err != nil {
		return false, err.Error()
	}

	writer.WriteField("path", path)

	err = writer.Close()

	if err != nil {
		return false, err.Error()
	}

	request, err := http.NewRequest("POST", "http://localhost:8080/Upload", body)

	if err != nil {
		return false, err.Error()
	}

	request.Header.Set("Content-Type", "multipart/form-data; boundary="+writer.Boundary())

	return testRequest(request, expected)
}

func testDownload(t *testing.T, file, source string) {
	if downloadAndHash(file) != hashFile(source) {
		t.Error(fmt.Sprintf("Downloaded file (%s) hash differs from source file (%s)", file, source))
	}
}

func testUpload(t *testing.T, file, path, expected string) {
	if ok, err := upload(file, path, expected); !ok {
		t.Error(err)
	}
}

func testRequest(request *http.Request, expected string) (bool, string) {
	if session != "" {
		request.AddCookie(&http.Cookie{Name: "session", Value: session})
	}

	res, err := http.DefaultClient.Do(request)

	if err != nil {
		return false, err.Error()
	}

	if res.StatusCode != 200 {
		return false, fmt.Sprintf("Expected response code 200 but got %s", res.Status)
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

	for _, v := range res.Cookies() {
		if v.Name == "session" {
			session = v.Value
		}
	}

	return true, ""
}

func sendPOST(service, params, expected string) (bool, string) {
	request, err := http.NewRequest("POST", "http://localhost:8080/"+service, strings.NewReader(params))

	if err != nil {
		return false, err.Error()
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return testRequest(request, expected)
}

func testPOST(t *testing.T, service, params, expected string) {
	if ok, err := sendPOST(service, params, expected); !ok {
		t.Error(err)
	}
}

func testFailPOST(t *testing.T, service, params string) {
	if ok, _ := sendPOST(service, params, ""); ok {
		t.Error("Test should have failed but succeeded")
	}
}

func TestArgs(t *testing.T) {
	testFailPOST(t, "Login", "password="+miogo.conf.AdminPassword)
	testFailPOST(t, "Login", "email="+miogo.conf.AdminEmail)
	testFailPOST(t, "Login", "password=XXX"+miogo.conf.AdminPassword)
	testFailPOST(t, "Login", "email=XXX"+miogo.conf.AdminEmail)
	testFailPOST(t, "Login", "passw&ord=garba&ge")
}

func TestLogin(t *testing.T) {
	testPOST(t, "Login", fmt.Sprintf("email=%sXXX&password=%s", miogo.conf.AdminEmail, miogo.conf.AdminPassword), `{ "success": "false" }`)
	testPOST(t, "Login", fmt.Sprintf("email=%s&password=%sXXX", miogo.conf.AdminEmail, miogo.conf.AdminPassword), `{ "success": "false" }`)

	if session != "" {
		t.Error("Session cookie should not have been returned by the server")
	}

	testPOST(t, "Login", fmt.Sprintf("email=%s&password=%s", miogo.conf.AdminEmail, miogo.conf.AdminPassword), `{ "success": "true" }`)

	if session == "" {
		t.Error("Session cookie has not been returned by the server")
	}
}

func TestNewFolder(t *testing.T) {
	testPOST(t, "NewFolder", "path=/test/test", `{ "error": "Bad folder name" }`)
	testPOST(t, "NewFolder", "path=/test", `{ "success": "true" }`)
	testPOST(t, "NewFolder", "path=/test/test", `{ "success": "true" }`)
	testPOST(t, "NewFolder", "path=/test/test", `{ "error": "Folder already exists" }`)
}

func TestUpload(t *testing.T) {
	testUpload(t, "README.md", "/test", `{ "success": "true" }`)
	// TODO: test multiple files upload
	testUpload(t, "main.go", "/test/a/b", `{ "error": "Wrong path" }`)
}

func TestGetFile(t *testing.T) {
	testDownload(t, "/test/README.md", "README.md")
}

func TestUser(t *testing.T) {
	testPOST(t, "NewUser", "email=test@miogo.tld&password=test", `{ "success": "true" }`)
	testPOST(t, "NewUser", "email=test2@miogo.tld&password=test", `{ "success": "true" }`)
	testPOST(t, "NewUser", "email=test3@miogo.tld&password=test", `{ "success": "true" }`)
	testPOST(t, "RemoveUser", "email=test3@miogo.tld", `{ "success": "true" }`)
	testPOST(t, "NewUser", "email=test@miogo.tld&password=1234", `{ "error": "user already exists" }`)
	// TODO: List users
}

func TestGroup(t *testing.T) {
	testPOST(t, "NewGroup", "name=miogo", `{ "success": "true" }`)
	testPOST(t, "NewGroup", "name=test", `{ "success": "true" }`)
	testPOST(t, "NewGroup", "name=miogo", `{ "error": "Group already exists" }`)

	// TODO : check if user exists before adding/removing them
	testPOST(t, "AddUserToGroup", "group=miogo&user=test@miogo.tld", `{ "success": "true" }`)
	testPOST(t, "AddUserToGroup", "group=miogo&user=test2@miogo.tld", `{ "success": "true" }`)
	testPOST(t, "RemoveUserFromGroup", "group=miogo&user=test2@miogo.tld", `{ "success": "true" }`)

	testPOST(t, "AddUserToGroup", "group=test&user=test@miogo.tld", `{ "success": "true" }`)
	testPOST(t, "AddUserToGroup", "group=test&user=test2@miogo.tld", `{ "success": "true" }`)
	testPOST(t, "RemoveGroup", "name=test", `{ "success": "true" }`)

	testPOST(t, "SetGroupAdmin", "group=miogo&user=test@miogo.tld", `{ "success": "true" }`)
}

func TestSetRights(t *testing.T) {
	testPOST(t, "SetResourceRights", "resource=/&user=test@miogo.tld&rights=rw", `{ "success": "true" }`)
	//testPOST(t, "SetResourceRights", "resource=/test/README.md&user=test@miogo.tld&rights=rw", `{ "success": "true" }`)

	testPOST(t, "SetResourceRights", "resource=/&group=miogo&rights=w", `{ "success": "true" }`)
	//testPOST(t, "SetResourceRights", "resource=/test/README.md&group=miogo&rights=w", `{ "success": "true" }`)

	testPOST(t, "SetResourceRights", "resource=/&rights=r&all=", `{ "success": "true" }`)
	//testPOST(t, "SetResourceRights", "resource=/test/README.md&rights=r&all=", `{ "success": "true" }`)
}

func TestGetFolder(t *testing.T) {
	testPOST(t, "GetFolder", "path=/test/test", `{"path":"/test/test","rights":{"all":"r","groups":[{"name":"miogo","rights":"w"}],"users":[{"name":"test@miogo.tld","rights":"rw"}]}}`)
	testPOST(t, "GetFolder", "path=/test", `{"path":"/test","files":[{"name":"README.md","rights":{"all":"r","groups":[{"name":"miogo","rights":"w"}],"users":[{"name":"test@miogo.tld","rights":"rw"}]}}],"folders":[{"path":"/test/test"}],"rights":{"all":"r","groups":[{"name":"miogo","rights":"w"}],"users":[{"name":"test@miogo.tld","rights":"rw"}]}}`)
	testPOST(t, "GetFolder", "path=/", `{"path":"/","folders":[{"path":"/test"}],"rights":{"all":"r","groups":[{"name":"miogo","rights":"w"}],"users":[{"name":"test@miogo.tld","rights":"rw"}]}}`)
}

func TestLogout(t *testing.T) {
	testPOST(t, "Logout", "", `{ "success": "true" }`)

	if session != "" {
		t.Error("Session cookie should have been cleared")
	}

	testFailPOST(t, "Logout", "")
}
