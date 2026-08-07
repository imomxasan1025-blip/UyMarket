package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "UyMarket bot is running!")
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}

	http.HandleFunc("/", homeHandler)

	log.Printf("UyMarket started on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
