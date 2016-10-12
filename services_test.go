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
	testPOST(t, "Login", fmt.Sprintf("email=%sXXX&password=%s", miogo.conf.AdminEmail, miogo.conf.AdminPassword), jsonkv("error", "User does not exist"))
	testPOST(t, "Login", fmt.Sprintf("email=%s&password=%sXXX", miogo.conf.AdminEmail, miogo.conf.AdminPassword), jsonkv("error", "Wrong password"))

	if session != "" {
		t.Error("Session cookie should not have been returned by the server")
	}

	testPOST(t, "Login", fmt.Sprintf("email=%s&password=%s", miogo.conf.AdminEmail, miogo.conf.AdminPassword), jsonkv("success", "true"))

	if session == "" {
		t.Error("Session cookie has not been returned by the server")
	}
}

func TestNewFolder(t *testing.T) {
	testPOST(t, "NewFolder", "path=/test/test", jsonkv("error", "Bad folder name"))
	testPOST(t, "NewFolder", "path=/test", jsonkv("success", "true"))
	testPOST(t, "NewFolder", "path=/test/test", jsonkv("success", "true"))
	testPOST(t, "NewFolder", "path=/test/test", jsonkv("error", "Folder already exists"))
}

func TestUpload(t *testing.T) {
	testUpload(t, "README.md", "/test", jsonkv("success", "true"))
	// TODO: test multiple files upload
	testUpload(t, "main.go", "/test/a/b", jsonkv("error", "Wrong path"))
}

func TestGetFile(t *testing.T) {
	testDownload(t, "/test/README.md", "README.md")
}

func TestUser(t *testing.T) {
	testPOST(t, "NewUser", "email=test@miogo.tld&password=test", jsonkv("success", "true"))
	testPOST(t, "NewUser", "email=test2@miogo.tld&password=test", jsonkv("success", "true"))
	testPOST(t, "NewUser", "email=test3@miogo.tld&password=test", jsonkv("success", "true"))
	testPOST(t, "RemoveUser", "email=test3@miogo.tld", jsonkv("success", "true"))
	testPOST(t, "NewUser", "email=test@miogo.tld&password=1234", jsonkv("error", "User already exists"))
	// TODO: List users
}

func TestGroup(t *testing.T) {
	testPOST(t, "NewGroup", "name=miogo", jsonkv("success", "true"))
	testPOST(t, "NewGroup", "name=test", jsonkv("success", "true"))
	testPOST(t, "NewGroup", "name=miogo", jsonkv("error", "Group already exists"))

	// TODO : check if user exists before adding/removing them
	testPOST(t, "AddUserToGroup", "group=miogo&user=test2@miogo.tld", jsonkv("success", "true"))
	testPOST(t, "RemoveUserFromGroup", "group=miogo&user=test2@miogo.tld", jsonkv("success", "true"))
	testPOST(t, "AddUserToGroup", "group=test&user=test@miogo.tld", jsonkv("success", "true"))
	testPOST(t, "AddUserToGroup", "group=miogo&user=test@miogo.tld", jsonkv("success", "true"))
	testPOST(t, "AddUserToGroup", "group=test&user=test2@miogo.tld", jsonkv("success", "true"))
	testPOST(t, "RemoveGroup", "name=test", jsonkv("success", "true"))
}

func TestSetRights(t *testing.T) {
	testPOST(t, "SetResourceRights", "resource=/&rights=rw&all=", jsonkv("success", "true"))
	testPOST(t, "SetResourceRights", "resource=/&rights=rw&group=miogo", jsonkv("success", "true"))
	testPOST(t, "SetResourceRights", "resource=/test&rights=n&all=", jsonkv("success", "true"))
}

func TestGetFolder(t *testing.T) {
	testPOST(t, "GetFolder", "path=/", `{"path":"/","folders":[{"path":"/test"}],"rights":{"all":"rw","groups":[{"name":"miogo","rights":"rw"}]}}`)
}

func TestRightsVerification(t *testing.T) {
	testPOST(t, "GetFile", "path=/test/README.md", jsonkv("error", "Access denied"))
	testPOST(t, "GetFolder", "path=/test", jsonkv("error", "Access denied"))
	testUpload(t, "main.go", "/test", jsonkv("error", "Access denied"))
}

func TestRemoveFile(t *testing.T) {
	testUpload(t, "README.md", "/", jsonkv("success", "true"))
	testPOST(t, "Remove", "path=/README.md", jsonkv("success", "true"))
	// TODO:  add a test with GetFolder
}

func TestRemoveFolder(t *testing.T) {
	testPOST(t, "Remove", "path=/test", jsonkv("success", "true"))
	// TODO: add a test with GetFolder
}

func TestCopyFile(t *testing.T) {
	testUpload(t, "README.md", "/", jsonkv("success", "true"))
	testPOST(t, "NewFolder", "path=/test", jsonkv("success", "true"))
	testPOST(t, "Copy", "path=/README.md&destination=/test&destFilename=fichiercopie.md", jsonkv("success", "true"))
}

func TestCopyFolder(t *testing.T) {
	testPOST(t, "Copy", "path=/README.md&destination=/test&destFilename=fichiercopie2.md", jsonkv("success", "true"))
	testPOST(t, "Copy", "path=/README.md&destination=/test&destFilename=fichiercopie3.md", jsonkv("success", "true"))
	testPOST(t, "NewFolder", "path=/test/sousdossier", jsonkv("success", "true"))
	testPOST(t, "Copy", "path=/test&destination=/dossiercopie", jsonkv("success", "true"))
	testPOST(t, "Remove", "path=/test", jsonkv("success", "true"))
}

func TestLogout(t *testing.T) {
	testPOST(t, "Logout", "", jsonkv("success", "true"))

	if session != "" {
		t.Error("Session cookie should have been cleared")
	}

	testFailPOST(t, "Logout", "")
}
