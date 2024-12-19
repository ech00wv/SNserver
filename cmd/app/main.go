package main

import (
	"log"
	"net/http"

	"github.com/ech00wv/SNserver/internal/config"
	handler "github.com/ech00wv/SNserver/internal/handlers"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatalf("failed loading enviroment: %s", err)
	}

	apiCfg := config.InitializeApiConfig()

	serveMux := handler.InitializeMux(apiCfg)

	httpServer := http.Server{
		Addr:    ":8080",
		Handler: serveMux,
	}
	err = httpServer.ListenAndServe()
	if err != nil {
		log.Fatalf("Server failed: %s", err)
	}

}
