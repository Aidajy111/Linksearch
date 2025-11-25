package main

import (
	"fmt"
	"net/http"
	"os"

	"link-search/internal/handlers"

	"github.com/go-chi/chi/v5"
)

func main() {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8888"
	}

	r := chi.NewRouter()

	r.Get("/", handlers.MainHandler)
	r.Post("/links", handlers.CreateLinksHandler)
	r.Get("/report", handlers.ReportHandler)

	fmt.Printf("The server is running on the port: -- http://localhost:%s --\n", port)
	err := http.ListenAndServe(":"+port, r)
	if err != nil {
		panic(err)
	}
}
