package main

import (
	"log"
	"net/http"
)

func main() {
	loadDotEnv(".env")
	addr := getenv("PORT", "8080")
	log.Printf("xendit-api-mock listening on :%s", addr)

	s := newServer()
	registerRoutes(s)

	if err := http.ListenAndServe(":"+addr, nil); err != nil {
		log.Fatal(err)
	}
}
