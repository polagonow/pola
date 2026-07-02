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
	log.Fatal(http.ListenAndServe(pola.Addr(), nil))
}
