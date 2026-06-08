package main

import (
	"log"
	"net/http"

	"github.com/polagonow/pola"
)

func main() {
	if err := pola.Ready(); err != nil {
		log.Fatal(err)
	}
	log.Printf("server-actions-demo listening on http://%s\n", pola.Addr())
	log.Fatal(http.ListenAndServe(pola.Addr(), nil))
}
