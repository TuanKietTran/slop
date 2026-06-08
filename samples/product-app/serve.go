// serve.go — tiny static file server for the product-app sample.
// Usage: go run serve.go
package main

import (
	"log"
	"net/http"
)

func main() {
	dir := "."
	addr := ":9090"
	log.Printf("serving %s on http://localhost%s", dir, addr)
	if err := http.ListenAndServe(addr, http.FileServer(http.Dir(dir))); err != nil {
		log.Fatal(err)
	}
}
