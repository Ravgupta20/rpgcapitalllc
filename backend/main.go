package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"

	"bytes"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
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

type S3Client struct {
	client *s3.Client
	bucket string
}

var bucketName string
var s3Client *S3Client

// Initialize S3 client with AWS configuration
func NewS3Client(bucketName string) (*S3Client, error) {

	// Load AWS configuration from environment/IAM role
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(cfg)

	return &S3Client{client: client, bucket: bucketName}, nil
}

// ReadFile reads a file from S3 and returns its content as bytes
func (s *S3Client) ReadFile(key string) ([]byte, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get object from S3
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object %s: %w", key, err)
	}
	defer result.Body.Close()

	// Read the entire body
	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read object body: %w", err)
	}

	return data, nil
}

// ListFiles lists files in S3 bucket with optional prefix
func (s *S3Client) ListFiles() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
	}

	result, err := s.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	var files []string
	for _, obj := range result.Contents {
		files = append(files, *obj.Key)
	}

	return files, nil
}

// Add this method to your S3Client struct
func (s *S3Client) WriteFile(key string, data []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("failed to put object %s: %w", key, err)
	}

	return nil
}

func loadSubmissions() FormSubmissions {
	file, err := s3Client.ReadFile("contacts.json")
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
	formData := FormData{
		Name:      r.PostFormValue("name"),
		Email:     r.PostFormValue("email"),
		Message:   r.PostFormValue("message"),
		Timestamp: time.Now(),
	}

	// Load existing submissions
	submissions := loadSubmissions()
	// Add new submission
	submissions.Submissions = append(submissions.Submissions, formData)
	// err := json.NewDecoder(r.Body).Decode(&data)
	// if err != nil {
	// 	http.Error(w, "Invalid JSON data", http.StatusBadRequest)
	// 	return
	// }
	jsonData, err := json.MarshalIndent(submissions, "", "    ")
	if err != nil {
		http.Error(w, "Error marshaling JSON", http.StatusInternalServerError)
		return
	}
	err = s3Client.WriteFile("contacts.json", jsonData)
	if err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		return
	}
}

func bucket(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head><title>S3 File Reader</title></head>
<body>
	<h1>S3 File Reader</h1>
	<p>Available endpoints:</p>
	<ul>
		<li><a href="/read_file?file=example.txt">/read?file=filename</a> - Read a specific file</li>
		<li><a href="/list">/list</a> - List all files</li>
		<li><a href="/list?prefix=folder/">/list?prefix=folder/</a> - List files with prefix</li>
		<li><a href="/exists?file=example.txt">/exists?file=filename</a> - Check if file exists</li>
		<li><a href="/stream?file=example.txt">/stream?file=filename</a> - Stream a file</li>
	</ul>
</body>
</html>
		`)
}

func bucket_list(w http.ResponseWriter, r *http.Request) {

	files, err := s3Client.ListFiles()
	if err != nil {
		log.Printf("Error listing files: %v", err)
		http.Error(w, fmt.Sprintf("Error listing files: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "Files in bucket %s", bucketName)
	fmt.Fprintf(w, ":\n\n")

	for _, file := range files {
		fmt.Fprintf(w, "%s\n", file)
	}
}

// Read file endpoint
func read_file(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("file")
	if filename == "" {
		http.Error(w, "file parameter is required", http.StatusBadRequest)
		return
	}

	content, err := s3Client.ReadFile(filename)
	if err != nil {
		log.Printf("Error reading file %s: %v", filename, err)
		http.Error(w, fmt.Sprintf("Error reading file: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "Content of %s:\n\n%s", filename, content)
}

func main() {

	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Get configuration from environment variables
	bucketName = os.Getenv("S3_BUCKET_NAME")
	if bucketName == "" {
		log.Fatal("S3_BUCKET_NAME environment variable is not set")
	}

	s3Client, err = NewS3Client(bucketName)
	if err != nil {
		log.Fatalf("Failed to initialize S3 client: %v", err)
	}
	// HTTP handlers
	http.HandleFunc("/bucket", bucket)
	http.HandleFunc("/bucket_list", bucket_list)
	http.HandleFunc("/read_file", read_file)
	fileServer := http.FileServer(http.Dir("./static"))
	http.Handle("/", fileServer)
	http.HandleFunc("/api/save_contact", save_contact)
	port := ":8080"
	fmt.Printf("Server listing on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))

}
