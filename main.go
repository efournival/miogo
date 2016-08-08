package main

import (
	"log"

	"github.com/valyala/fasthttp"
)

func main() {
	miogo := NewMiogo()
	log.Fatal(fasthttp.ListenAndServe(":8080", miogo.GetHandler()))
}
