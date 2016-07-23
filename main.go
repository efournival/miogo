package main

import (
	"log"
	"net/http"

	"github.com/facebookgo/grace/gracehttp"
)

func main() {
	miogo := NewMiogo()
	log.Fatal(gracehttp.Serve(&http.Server{Addr: ":8070", Handler: miogo.mux}))
}
