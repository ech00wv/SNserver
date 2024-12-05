package main

import "net/http"

func main() {
	serveMux := http.ServeMux{}
	httpServer := http.Server{
		Handler: &serveMux,
		Addr:    ":8080",
	}

	httpServer.ListenAndServe()
}
