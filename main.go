package main

import (
	"net/http"

	"github.com/facebookgo/grace/gracehttp"
)

func main() {
	miogo := NewMiogo()
	gracehttp.Serve(&http.Server{Addr: ":8080", Handler: miogo.mux})
}
