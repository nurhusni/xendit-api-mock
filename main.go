package main

import (
	"log"
	"net/http"

	"xendit-api-mock/internal/callback"
	"xendit-api-mock/internal/scenario"
	"xendit-api-mock/internal/service/disbursement"
	httptransport "xendit-api-mock/internal/transport/http"
)

func main() {
	loadDotEnv(".env")
	addr := getenv("PORT", "8080")
	log.Printf("[main] xendit-api-mock listening on :%s", addr)

	engine := scenario.NewEngine(loadScenario(getenv("SCENARIO_FILE", "")))
	randomStatus := getenv("RANDOM_STATUS", "true") == "true"
	engine.WithRandomStatus(randomStatus)
	callbackURL := getenv("CALLBACK_URL", "")
	callbackToken := getenv("CALLBACK_TOKEN", "")
	callbackClient := callback.NewClient(callbackURL, callbackToken, nil)
	userID := getenv("XENDIT_USER_ID", "user_mock")
	service := disbursement.NewService(engine, callbackClient, userID)
	handler := httptransport.NewHandler(service, callbackURL)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	serverHandler := withDisbursementDoubleSlashAlias(mux)

	if err := http.ListenAndServe(":"+addr, serverHandler); err != nil {
		log.Fatal(err)
	}
}

func withDisbursementDoubleSlashAlias(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/xendit//disbursements" {
			cloned := r.Clone(r.Context())
			urlCopy := *r.URL
			urlCopy.Path = "/xendit/disbursements"
			cloned.URL = &urlCopy
			next.ServeHTTP(w, cloned)
			return
		}

		next.ServeHTTP(w, r)
	})
}
