package config

import (
	"database/sql"
	"log"
	"os"
	"sync/atomic"

	"github.com/ech00wv/SNserver/internal/database"
)

type ApiConfig struct {
	FileserverHits atomic.Int64
	Queries        *database.Queries
	Platfrom       string
}

func InitializeApiConfig() *ApiConfig {
	apiCfg := &ApiConfig{
		FileserverHits: atomic.Int64{},
		Queries:        initializeDBQueries(),
		Platfrom:       os.Getenv("PLATFORM"),
	}
	return apiCfg
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