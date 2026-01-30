package main

import (
	"log"
	"net/http"
)

func main() {
	loadDotEnv(".env")
	addr := getenv("PORT", "8080")
	log.Printf("[main] xendit-api-mock listening on :%s", addr)

	s := newServer()
	mux := http.NewServeMux()
	registerRoutes(mux, s)
	handler := loggingMiddleware(mux)

	if err := http.ListenAndServe(":"+addr, handler); err != nil {
		log.Fatal(err)
	}
}
