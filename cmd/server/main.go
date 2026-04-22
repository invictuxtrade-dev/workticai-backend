package main

import (
	"fmt"
	"log"
	"net/http"

	httpapi "whatsapp-sales-os-enterprise/backend/internal/httpapi"
)

func main() {
	srv, err := httpapi.NewServer()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("🚀 backend running on http://localhost:8090")
	log.Fatal(http.ListenAndServe(":8090", srv.Router))
}