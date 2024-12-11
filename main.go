package main

import (
	"net/http"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {

	godotenv.Load()

	apiCfg := initializeApiConfig()

	serveMux := initializeMux(apiCfg)

	httpServer := http.Server{
		Addr:    ":8080",
		Handler: serveMux,
	}
	httpServer.ListenAndServe()

}
