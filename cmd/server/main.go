package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	"spellingchallenge/internal/httpapi"
)

func main() {
	address := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		address = ":" + port
	}
	server := &http.Server{Addr: address, Handler: httpapi.NewFixture(nil)}
	log.Printf("spelling challenge listening on http://localhost%s", address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
