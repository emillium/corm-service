package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/couchbase/gocb/v2/search"
)

// Global configuration from environment variables
var (
	bucket      *gocb.Bucket
	cluster     *gocb.Cluster

	// The port the service listens on (standard for Kubernetes)
	PORT = os.Getenv("PORT") 
	// The clientState secret used for validating Graph webhooks
	CLIENT_STATE = os.Getenv("CLIENT_STATE")
	ftsIndexName = "mail_fts_index" // Matches the index name we'll create later
)

func init() {
	if PORT == "" {
		PORT = "8080"
	}
	if CLIENT_STATE == "" {
		log.Println("WARNING: CLIENT_STATE environment variable is not set. Webhook validation will be skipped.")
	}
}

func main() {
	// --- Couchbase Connection Setup ---
	connStr := os.Getenv("COUCHBASE_CONN_STR") 
	username := os.Getenv("COUCHBASE_USERNAME")
	password := os.Getenv("COUCHBASE_PASSWORD")
	bucketName := os.Getenv("COUCHBASE_BUCKET")

	if connStr == "" || username == "" || password == "" || bucketName == "" {
		log.Fatal("Missing Couchbase environment variables.")
	}

	var err error
	cluster, err = gocb.Connect(connStr, gocb.ClusterOptions{
		Username: username,
		Password: password,
	})
	if err != nil {
		log.Fatalf("Error connecting to Couchbase: %v", err)
	}

	bucket = cluster.Bucket(bucketName)

	// Wait for the bucket to be ready (critical in Kubernetes)
	err = bucket.WaitUntilReady(5*time.Second, nil)
	if err != nil {
		log.Fatalf("Bucket not ready: %v", err)
	}
	log.Printf("Successfully connected to Couchbase bucket: %s", bucketName)

	// --- HTTP Server Setup ---
	router := http.NewServeMux()
	router.HandleFunc("/api/mail/save", saveMailIngestHandler)
	router.HandleFunc("/api/mail/search", searchMailIngestHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting server on :%s", port)
	err = http.ListenAndServe(fmt.Sprintf(":%s", port), router)
	if err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// Handles incoming POST requests to save weather data
func saveMailIngestHandler(w http.ResponseWriter, r *http.Request) {
	// Ensure only POST requests are allowed
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// --- 1. Handle Graph Subscription Validation (Query Parameter Check) ---
	validationToken := r.URL.Query().Get("validationToken")
	if validationToken != "" {
		// Graph API requires a 200 OK with the token echoed back as plain text.
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(validationToken))
		log.Printf("INFO: Successfully handled Graph validation request. Token: %s", validationToken)
		return
	}

	// --- 2. Process Payload (VBA/PowerShell or Graph Event Notification) ---
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		log.Printf("ERROR: Failed to read body: %v", err)
		return
	}

	// Try to parse as a standardized payload (VBA/PowerShell direct ingest)
	var ingestPayload MailIngestPayload
	if err := json.Unmarshal(body, &ingestPayload); err == nil {
		// Successfully parsed a standardized payload (typically from PowerShell)
		
		// In a real application, you would check DataOrigin here and proceed to
		// process the mail metadata. For now, we log it.
		
		log.Printf("--- INGEST PAYLOAD RECEIVED (VBA/PowerShell) ---")
		log.Printf("MessageID: %s", ingestPayload.MessageID)
		log.Printf("Subject:   %s", ingestPayload.Subject)
		log.Printf("Origin:    %s", ingestPayload.DataOrigin)
		log.Println("-------------------------------------------------")
		
		// Return 200 OK for a successful synchronous ingestion
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "Ingested successfully"}`))
		return
	}

	// Try to parse as a Graph Notification (minimal data, requires a subsequent fetch)
	var notification GraphNotification
	if err := json.Unmarshal(body, &notification); err == nil && len(notification.Value) > 0 {
		// Successfully parsed a Graph notification
		
		// A. Validate Client State (Security Check)
		if CLIENT_STATE != "" && notification.Value[0].ClientState != CLIENT_STATE {
			log.Printf("SECURITY ALERT: ClientState mismatch! Received: %s, Expected: %s", notification.Value[0].ClientState, CLIENT_STATE)
			http.Error(w, "Client State Mismatch", http.StatusForbidden)
			return
		}
		
		// B. Log Event and Acknowledge (HTTP 202 Accepted)
		notificationResource := notification.Value[0].Resource
		notificationMessageID := notification.Value[0].ResourceData.ID

		log.Printf("--- GRAPH WEBHOOK EVENT RECEIVED ---")
		log.Printf("Resource: %s", notificationResource)
		log.Printf("Message ID to Fetch: %s", notificationMessageID)
		log.Println("------------------------------------")
		
		// In a real app: launch a separate goroutine/worker to perform the Graph API fetch 
		// and then send the result to the main processing queue.

		docID := fmt.Sprintf("%s_%s_%d", notificationResource, notificationMessageID, time.Now().UnixNano())

		_, err := bucket.DefaultCollection().Upsert(docID, notification, &gocb.UpsertOptions{
			DurabilityLevel: gocb.DurabilityLevelMajority,
		})

		if err != nil {
			log.Printf("Couchbase Upsert error: %v", err)
			http.Error(w, "Failed to save data: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Return 202 Accepted, as processing is asynchronous after acknowledgement.
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprintf(w, `{"status": "success", "id": "%s"}`, docID)
		return
	}

	// If neither parsing succeeded, log the error and respond with a bad request
	log.Printf("ERROR: Request body did not match standardized payload or Graph notification format. Body: %s", string(body))
	http.Error(w, "Invalid payload format", http.StatusBadRequest)
}

// Handles incoming GET requests to search weather data
func searchMailIngestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	queryStr := r.URL.Query().Get("q")
	if queryStr == "" {
		http.Error(w, "Missing search query parameter 'q'", http.StatusBadRequest)
		return
	}

	// Create a MatchQuery for Full-Text Search
	ftsQuery := search.NewMatchQuery(queryStr)
	
	// Execute the FTS query against the cluster
	results, err := cluster.SearchQuery(
		ftsIndexName, 
		ftsQuery,
		&gocb.SearchOptions{
			Limit: 10, // Limit results
			Fields: []string{"messageId", "sourceAccount", "senderEmail", "subject", "dataOrigin"}, // Return specific fields
		},
	)

	if err != nil {
		log.Printf("Couchbase Search error: %v", err)
		http.Error(w, "Search failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare results for response
	var hits []map[string]interface{}
	for results.Next() {
		hit := results.Row()
		hits = append(hits, map[string]interface{}{
			"id": hit.ID,
			"score": hit.Score,
			"fields": hit.Fields, // Contains the requested fields
		})
	}

	if err := results.Err(); err != nil {
		log.Printf("FTS iteration error: %v", err)
		http.Error(w, "Error processing search results", http.StatusInternalServerError)
		return
	}

	meta, err := results.MetaData()
	if err != nil {
		// Log and handle error if metadata fetch fails, though unlikely after a successful query.
		log.Printf("FTS metadata error: %v", err)
	}

	// Now access TotalHits via the correctly retrieved meta object.
	totalHits := meta.Metrics.TotalRows

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"query": queryStr,
		"total_hits": totalHits,
		"results": hits,
	})
}
