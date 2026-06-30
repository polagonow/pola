package main

import (
	"fmt"
	"os"

	"validation/repositories"

	"github.com/polagonow/pola/validation"
)

func main() {
	valid := repositories.Contact{
		Name:    "Alice",
		Email:   "alice@example.com",
		Website: "https://alice.dev",
		Phone:   "5551234567",
	}
	if err := validation.Validate(&valid); err != nil {
		fmt.Fprintf(os.Stderr, "unexpected error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("valid contact: OK")

	invalid := repositories.Contact{
		Name:  "Alice123",     // not alphabetic
		Email: "not-an-email", // not a valid email
		Phone: "not-numeric",  // not digits
	}
	if err := validation.Validate(&invalid); err != nil {
		fmt.Printf("invalid contact caught: %v\n", err)
	}

	server := repositories.Server{
		Hostname:   "webserver",
		IpAddress:  "192.168.1.1",
		MacAddress: "01:23:45:67:89:ab",
		Port:       "8080",
		Version:    "1.2.3",
	}
	if err := validation.Validate(&server); err != nil {
		fmt.Fprintf(os.Stderr, "unexpected error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("valid server:  OK")
}
