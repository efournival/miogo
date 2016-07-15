package main

import (
	"github.com/facebookgo/grace/gracehttp"
	"net/http"
)

func main() {
	miogo := NewMiogo()
	gracehttp.Serve(&http.Server{Addr: ":8080", Handler: miogo.mux})
}
