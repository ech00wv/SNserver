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
	serveMux.HandleFunc("GET /api/status", handleStatus)
	serveMux.HandleFunc("POST /api/users", ah.createUser)
	serveMux.HandleFunc("PUT /api/users", ah.updateUser)
	serveMux.HandleFunc("POST /api/messages", ah.createMessage)
	serveMux.HandleFunc("GET /api/messages", ah.getAllMessages)
	serveMux.HandleFunc("GET /api/messages/{messageID}", ah.getMessage)
	serveMux.HandleFunc("POST /api/login", ah.loginUser)
	serveMux.HandleFunc("POST /api/refresh", ah.refreshAccessToken)
	serveMux.HandleFunc("POST /api/revoke", ah.revokeRefreshToken)
	serveMux.HandleFunc("DELETE /api/messages/{messageID}", ah.deleteMessage)
	serveMux.HandleFunc("POST /api/payment/webhook", ah.proceedPayment)
	return serveMux
}

func respondWithError(rw http.ResponseWriter, code int, errorMessage string) {

	rw.Header().Set("Content-Type", "application/json")

	rw.WriteHeader(code)

	responseError := struct {
		Err string `json:"error"`
	}{
		Err: errorMessage,
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

// handle server status on GET /status
func handleStatus(rw http.ResponseWriter, req *http.Request) {
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

	userService := service.UserService{ApiConfig: ah.ApiCfg}
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

	userService := service.UserService{ApiConfig: ah.ApiCfg}
	var reqBodyData models.UserRequest

	err := json.NewDecoder(req.Body).Decode(&reqBodyData)
	defer req.Body.Close()
	if err != nil {
		respondWithError(rw, http.StatusBadRequest, fmt.Sprintf("error marshalling json: %s", err))
		return
	}

	user, status, err := userService.CreateUser(req.Context(), reqBodyData)
	if err != nil {
		respondWithError(rw, status, err.Error())
		return
	}

	respondWithJson(rw, status, user)
}

// Create message on POST /api/messages
func (ah *ApiHandler) createMessage(rw http.ResponseWriter, req *http.Request) {
	messageService := service.MessageService{ApiConfig: ah.ApiCfg}

	var reqBodyData models.MessageRequest
	err := json.NewDecoder(req.Body).Decode(&reqBodyData)
	defer req.Body.Close()
	if err != nil {
		respondWithError(rw, http.StatusInternalServerError, err.Error())
		return
	}
	message, status, err := messageService.CreateMessage(req.Context(), req.Header, reqBodyData)
	if err != nil {
		respondWithError(rw, status, err.Error())
		return
	}
	respondWithJson(rw, status, message)

}

func (ah *ApiHandler) getAllMessages(rw http.ResponseWriter, req *http.Request) {
	messageService := service.MessageService{ApiConfig: ah.ApiCfg}
	messages, status, err := messageService.GetAllMessages(req.Context())

	if err != nil {
		respondWithError(rw, status, err.Error())
		return
	}

	respondWithJson(rw, status, messages)
}

func (ah *ApiHandler) getMessage(rw http.ResponseWriter, req *http.Request) {
	messageService := service.MessageService{ApiConfig: ah.ApiCfg}

	messageID := req.PathValue("messageID")
	message, status, err := messageService.GetMessage(req.Context(), messageID)
	if err != nil {
		respondWithError(rw, status, err.Error())
		return
	}
	respondWithJson(rw, status, message)
}

func (ah *ApiHandler) loginUser(rw http.ResponseWriter, req *http.Request) {
	var reqBodyData models.UserRequest

	err := json.NewDecoder(req.Body).Decode(&reqBodyData)
	defer req.Body.Close()
	if err != nil {
		respondWithError(rw, http.StatusInternalServerError, fmt.Sprintf("cannot decode user: %s", err))
		return
	}
	userService := service.UserService{ApiConfig: ah.ApiCfg}

	user, status, err := userService.LoginUser(req.Context(), reqBodyData)
	if err != nil {
		respondWithError(rw, status, err.Error())
		return
	}

	respondWithJson(rw, status, user)

}

func (ah *ApiHandler) refreshAccessToken(rw http.ResponseWriter, req *http.Request) {
	tokenServ := service.TokenService{Queries: ah.ApiCfg.Queries}
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
	tokenServ := service.TokenService{Queries: ah.ApiCfg.Queries}
	status, err := tokenServ.RevokeRefreshToken(req.Context(), req.Header)
	if err != nil {
		respondWithError(rw, status, fmt.Sprintf("cannot revoke token: %s", err))
		return
	}
	respondWithJson(rw, status, nil)
}

func (ah *ApiHandler) updateUser(rw http.ResponseWriter, req *http.Request) {
	var reqBodyData models.UserRequest

	err := json.NewDecoder(req.Body).Decode(&reqBodyData)
	if err != nil {
		respondWithError(rw, http.StatusInternalServerError, "cannot decode json")
		return
	}

	userServ := service.UserService{ApiConfig: ah.ApiCfg}

	dbUser, status, err := userServ.UpdateUser(req.Context(), req.Header, reqBodyData.Email, reqBodyData.Password)
	if err != nil {
		respondWithError(rw, status, fmt.Sprintf("cannot update user: %s", err))
		return
	}

	respondWithJson(rw, status, dbUser)
}

func (ah *ApiHandler) deleteMessage(rw http.ResponseWriter, req *http.Request) {
	messageServ := service.MessageService{ApiConfig: ah.ApiCfg}
	messageID := req.PathValue("messageID")

	status, err := messageServ.DeleteMessage(req.Context(), req.Header, messageID)
	if err != nil {
		respondWithError(rw, status, fmt.Sprintf("error in message deletion: %s", err))
		return
	}

	respondWithJson(rw, status, nil)
}

func (ah *ApiHandler) proceedPayment(rw http.ResponseWriter, req *http.Request) {
	var reqBodyData models.PaymentProviderWebhook
	err := json.NewDecoder(req.Body).Decode(&reqBodyData)
	if err != nil {
		rw.WriteHeader(http.StatusNoContent)
		return
	}

	reqHeader := req.Header
	paymentServ := service.PaymentService{ApiConfig: ah.ApiCfg}
	status := paymentServ.UpgradeToPremium(req.Context(), reqBodyData, reqHeader)
	rw.WriteHeader(status)
	return
}
