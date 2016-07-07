package main

import (
	"net/http"
	"golang.org/x/net/webdav"
	"time"
)

func main() {
	handler := webdav.Handler{
		FileSystem: webdav.Dir("/home/efournival/Bureau/test/"),
		LockSystem: webdav.NewMemLS(),
		Logger: func(r *http.Request, e error) {
			time.Sleep(15 * time.Millisecond)
		},
	}

	http.HandleFunc("/", handler.ServeHTTP)
	http.ListenAndServe(":8080", nil)
}
