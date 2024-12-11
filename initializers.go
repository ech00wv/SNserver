package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/ech00wv/SNserver/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int64
	queries        *database.Queries
	platfrom       string
}

func initializeApiConfig() *apiConfig {
	apiCfg := &apiConfig{
		fileserverHits: atomic.Int64{},
		queries:        initializeDBQueries(),
		platfrom:       os.Getenv("PLATFORM"),
	}
	return apiCfg
}

// initialize router
func initializeMux(apiCfg *apiConfig) *http.ServeMux {
	serveMux := http.NewServeMux()

	serveMux.Handle("/app/", apiCfg.middlewareMetrics(
		http.StripPrefix("/app", http.FileServer(http.Dir("."))),
	))

	serveMux.HandleFunc("GET /admin/metrics", apiCfg.serveMetrics)
	serveMux.HandleFunc("POST /admin/reset", apiCfg.resetApp)
	serveMux.HandleFunc("GET /api/healthz", handleHealthz)
	serveMux.HandleFunc("POST /api/messages", apiCfg.createMessage)
	serveMux.HandleFunc("POST /api/users", apiCfg.createUser)
	serveMux.HandleFunc("GET /api/messages", apiCfg.getMessages)
	serveMux.HandleFunc("GET /api/messages/{messageId}", apiCfg.getMessage)

	return serveMux
}

func initializeDBQueries() *database.Queries {
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Printf("error in db connection: %s", err)
		return nil
	}
	dbQueries := database.New(db)
	return dbQueries
}
