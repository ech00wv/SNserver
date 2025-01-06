package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/ech00wv/SNserver/internal/config"
	"github.com/ech00wv/SNserver/internal/models"
	service "github.com/ech00wv/SNserver/internal/services"
)

type ApiHandler struct {
	ApiCfg *config.ApiConfig
}

func InitializeMux(ac *config.ApiConfig) *http.ServeMux {

	ah := &ApiHandler{ApiCfg: ac}
	serveMux := http.NewServeMux()

	serveMux.Handle("/app/", ah.middlewareMetrics(
		http.StripPrefix("/app", http.FileServer(http.Dir("../../assets"))),
	))

	serveMux.HandleFunc("GET /admin/metrics", ah.serveMetrics)
	serveMux.HandleFunc("POST /admin/reset", ah.resetApp)
	serveMux.HandleFunc("GET /api/healthz", handleHealthz)
	serveMux.HandleFunc("POST /api/messages", ah.createMessage)
	serveMux.HandleFunc("POST /api/users", ah.createUser)
	serveMux.HandleFunc("GET /api/messages", ah.getAllMessages)
	serveMux.HandleFunc("GET /api/messages/{messageId}", ah.getMessage)
	serveMux.HandleFunc("POST /api/login", ah.loginUser)
	serveMux.HandleFunc("POST /api/refresh", ah.refreshAccessToken)
	serveMux.HandleFunc("POST /api/revoke", ah.revokeRefreshToken)
	return serveMux
}

func respondWithError(rw http.ResponseWriter, code int, msg string) {

	rw.Header().Set("Content-Type", "application/json")

	rw.WriteHeader(code)

	responseError := struct {
		Err string `json:"error"`
	}{
		Err: msg,
	}
	jsonErr, _ := json.Marshal(responseError)

	rw.Write(jsonErr)
}

// respond with json of given payload structure and responding with it to request
func respondWithJson(rw http.ResponseWriter, code int, payload interface{}) {

	rw.Header().Set("Content-Type", "application/json")

	encodedJson, err := json.Marshal(payload)
	if err != nil {
		respondWithError(rw, http.StatusInternalServerError, fmt.Sprintf("error marshalling json: %s", err))
		return
	}

	rw.WriteHeader(code)
	rw.Write(encodedJson)
}

// handle server status on GET /healthz
func handleHealthz(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	rw.WriteHeader(200)
	rw.Write([]byte("OK"))
}

// middleware for counting number of requests to GET /app URL
func (ah *ApiHandler) middlewareMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		ah.ApiCfg.FileserverHits.Add(1)
		next.ServeHTTP(rw, req)
	})
}

// handler to see how many requests was sent on GET /app URL by GET /admin/metrics
func (ah *ApiHandler) serveMetrics(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	rw.WriteHeader(200)
	fmt.Fprintf(rw, `<html>
					<body>
						<h1>Welcome, Admin</h1>
						<p>Social Network has been visited %d times!</p>
					</body>
				</html>`, ah.ApiCfg.FileserverHits.Load())
}

// reset whole app on POST /admin/reset
func (ah *ApiHandler) resetApp(rw http.ResponseWriter, req *http.Request) {
	if ah.ApiCfg.Platfrom != "dev" {
		rw.WriteHeader(http.StatusForbidden)
		return
	}

	userService := service.UserService{Queries: ah.ApiCfg.Queries}
	err := userService.DeleteUsers(req.Context())
	if err != nil {
		respondWithError(rw, http.StatusInternalServerError, fmt.Sprintf("error in deleting users: %s", err))
		return
	}

	ah.ApiCfg.FileserverHits = atomic.Int64{}
	rw.Header().Set("Content-Type", "text/plain; charset=utf8")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte("app successfuly resetted!"))

}

// create a user on POST /api/users
func (ah *ApiHandler) createUser(rw http.ResponseWriter, req *http.Request) {

	userService := service.UserService{Queries: ah.ApiCfg.Queries}
	var reqBody models.UserRequest

	err := json.NewDecoder(req.Body).Decode(&reqBody)
	defer req.Body.Close()
	if err != nil {
		respondWithError(rw, http.StatusBadRequest, fmt.Sprintf("error marshalling json: %s", err))
		return
	}

	user, status, err := userService.CreateUser(req.Context(), reqBody)
	if err != nil {
		respondWithError(rw, status, err.Error())
		return
	}

	respondWithJson(rw, status, user)
}

// Create message on POST /api/messages
func (ah *ApiHandler) createMessage(rw http.ResponseWriter, req *http.Request) {
	messageService := service.MessageService{Queries: ah.ApiCfg.Queries}

	var reqBody models.MessageRequest
	err := json.NewDecoder(req.Body).Decode(&reqBody)
	defer req.Body.Close()
	if err != nil {
		respondWithError(rw, http.StatusInternalServerError, err.Error())
		return
	}
	message, status, err := messageService.CreateMessage(req.Context(), req.Header, reqBody, ah.ApiCfg)
	if err != nil {
		respondWithError(rw, status, err.Error())
		return
	}
	respondWithJson(rw, status, message)

}

func (ah *ApiHandler) getAllMessages(rw http.ResponseWriter, req *http.Request) {
	messageService := service.MessageService{Queries: ah.ApiCfg.Queries}
	messages, status, err := messageService.GetAllMessages(req.Context())

	if err != nil {
		respondWithError(rw, status, err.Error())
		return
	}

	respondWithJson(rw, status, messages)
}

func (ah *ApiHandler) getMessage(rw http.ResponseWriter, req *http.Request) {
	messageService := service.MessageService{Queries: ah.ApiCfg.Queries}

	messageId := req.PathValue("messageId")
	message, status, err := messageService.GetMessage(req.Context(), messageId)
	if err != nil {
		respondWithError(rw, status, err.Error())
		return
	}
	respondWithJson(rw, status, message)
}

func (ah *ApiHandler) loginUser(rw http.ResponseWriter, req *http.Request) {
	var reqBody models.UserRequest

	err := json.NewDecoder(req.Body).Decode(&reqBody)
	defer req.Body.Close()
	if err != nil {
		respondWithError(rw, http.StatusInternalServerError, fmt.Sprintf("cannot decode user: %s", err))
		return
	}
	userService := service.UserService{Queries: ah.ApiCfg.Queries}

	user, status, err := userService.LoginUser(req.Context(), reqBody, ah.ApiCfg)
	if err != nil {
		respondWithError(rw, status, err.Error())
		return
	}

	respondWithJson(rw, status, user)

}

func (ah *ApiHandler) refreshAccessToken(rw http.ResponseWriter, req *http.Request) {
	tokenServ := service.TokenService{Queries: *ah.ApiCfg.Queries}
	newToken, status, err := tokenServ.RefreshAccessToken(req.Context(), req.Header, ah.ApiCfg)
	if err != nil {
		respondWithError(rw, status, fmt.Sprintf("error in token refreshing: %s", err))
		return
	}
	jsonToken := struct {
		Token string `json:"token"`
	}{
		Token: newToken,
	}
	respondWithJson(rw, status, jsonToken)
}

func (ah *ApiHandler) revokeRefreshToken(rw http.ResponseWriter, req *http.Request) {
	tokenServ := service.TokenService{Queries: *ah.ApiCfg.Queries}
	status, err := tokenServ.RevokeRefreshToken(req.Context(), req.Header)
	if err != nil {
		respondWithError(rw, status, fmt.Sprintf("cannot revoke token: %s", err))
		return
	}
	respondWithJson(rw, status, nil)
}
