package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type FormData struct {
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type FormSubmissions struct {
	Submissions []FormData `json:"submissions"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	http.FileServer(http.Dir("./static"))
}

func loadSubmissions() FormSubmissions {
	file, err := os.ReadFile("contacts.json")
	if err != nil {
		return FormSubmissions{Submissions: []FormData{}}
	}

	var submissions FormSubmissions
	err = json.Unmarshal(file, &submissions)
	if err != nil {
		log.Printf("Error unmarshaling JSON: %v", err)
		return FormSubmissions{Submissions: []FormData{}}
	}

	return submissions
}
func save_contact(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract form fields
	data := FormData{
		Name:      r.PostFormValue("name"),
		Email:     r.PostFormValue("email"),
		Message:   r.PostFormValue("message"),
		Timestamp: time.Now(),
	}

	// Load existing submissions
	submissions := loadSubmissions()
	// err := json.NewDecoder(r.Body).Decode(&data)
	// if err != nil {
	// 	http.Error(w, "Invalid JSON data", http.StatusBadRequest)
	// 	return
	// }
	jsonData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		http.Error(w, "Error marshaling JSON", http.StatusInternalServerError)
		return
	}
	err = os.WriteFile("contacts.json", jsonData, 0644) // 0644 is file permission
	if err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}
}

func main() {
	fileServer := http.FileServer(http.Dir("./static"))
	http.Handle("/", fileServer)
	http.HandleFunc("/api/save_contact", save_contact)
	port := ":8080"
	fmt.Printf("Server listing on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))

}
