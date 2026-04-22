package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	httpapi "whatsapp-sales-os-enterprise/backend/internal/httpapi"
)

func main() {
	srv, err := httpapi.NewServer()
	if err != nil {
		log.Fatal(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	fmt.Printf("🚀 backend running on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, srv.Router))
}