package main

import (
	"fmt"
	"log"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, Go Server!")
}
func main() {
	http.HandleFunc("/", handler)
	port := ":8080"
	fmt.Printf("Server listing on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
