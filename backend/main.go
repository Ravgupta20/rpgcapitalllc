package main

import (
	"fmt"
	"log"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	http.FileServer(http.Dir("./static"))
}

func save_contact(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "save_contact")
}

func main() {
	fileServer := http.FileServer(http.Dir("./static"))
	http.Handle("/", fileServer)
	http.HandleFunc("/api/save_contact", save_contact)
	port := ":8080"
	fmt.Printf("Server listing on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))

}
