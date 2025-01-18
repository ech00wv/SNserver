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

type responseError struct {
	Err string `json:"error"`
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

	responseError := responseError{Err: errorMessage}
	jsonErr, _ := json.Marshal(responseError)

	rw.Write(jsonErr)
}

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

// @Summary Checking server status
// @Description Returns just an "OK"
// @Produce text/html
// @Success 200 {string} string "OK"
// @Router /status [get]
func handleStatus(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	rw.WriteHeader(200)
	rw.Write([]byte("OK"))
}

func (ah *ApiHandler) middlewareMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		ah.ApiCfg.FileserverHits.Add(1)
		next.ServeHTTP(rw, req)
	})
}

// @Summary Fileservers metrics
// @Description Returns an html with visitors counter
// @Produce text/html
// @Success 200 {string} string "html page with metrics"
// @Router /metrics [get]
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

// @Summary Reset app
// @Description Reset app and clear all the users (hence messages, etc.)
// @Success 200 {string} string "app successfully resetted!"
// @Failure 403
// @Failure 500 {object} handler.responseError "error in deleting users"
// @Router /admin/reset [post]
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

// @Summary User creation
// @Description Create a user with provided email and password
// @Accept  json
// @Produce json
// @Param email body string true "User's email"
// @Param password body string true "User's password"
// @Success 201 {object} models.UserResponse "Created user's information"
// @Failure 400 {object} handler.responseError "User credentials is incorrect"
// @Failure 500 {object} handler.responseError "Internal server error"
// @Router /api/users [post]
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

// @Summary Message creation
// @Description Create a message for given user
// @Accept  json
// @Produce json
// @Param body body string true "Message content"
// @Param Authorization header string true "Access token"
// @Success 201 {object} models.MessageResponse "Created message information"
// @Failure 400 {object} handler.responseError "Something is wrong in provided information"
// @Failure 401 {object} handler.responseError "User is unauthorized"
// @Failure 500 {object} handler.responseError "Internal server error"
// @Router /api/messages [post]
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

// @Summary Get all messages
// @Description Get all messages (either all of them or from specific author)
// @Produce json
// @Param author_id query string false "author_id"
// @Param sort query string false "Sorting order ('asc', 'desc' or nothing)"
// @Success 200 {array} models.MessageResponse "List of messages"
// @Failure 400 {object} handler.responseError "Something is wrong in provided information"
// @Failure 404 {object} handler.responseError "Messages not found"
// @Router /api/messages [get]
func (ah *ApiHandler) getAllMessages(rw http.ResponseWriter, req *http.Request) {
	messageService := service.MessageService{ApiConfig: ah.ApiCfg}
	authorID := req.URL.Query().Get("author_id")
	sortingOrder := req.URL.Query().Get("sort")
	messages, status, err := messageService.GetAllMessages(req.Context(), authorID, sortingOrder)

	if err != nil {
		respondWithError(rw, status, err.Error())
		return
	}

	respondWithJson(rw, status, messages)
}

// @Summary Get message
// @Description Get one specific message by it's id
// @Produce json
// @Param messageID path string true "messageID"
// @Success 200 {object} models.MessageResponse "Message content"
// @Failure 400 {object} handler.responseError "Something is wrong in provided information"
// @Failure 404 {object} handler.responseError "Message not found"
// @Failure 500 {object} handler.responseError "Internal server error"
// @Router /api/messages/{messageID} [get]
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

// @Summary Login user
// @Description Login user with email and password
// @Accept json
// @Produce json
// @Param email body string true "User's email"
// @Param password body string true "User's password"
// @Success 200 {object} models.UserResponse "User's data"
// @Failure 400 {object} handler.responseError "Something is wrong in provided information"
// @Failure 401 {object} handler.responseError "User is unauthorized"
// @Failure 500 {object} handler.responseError "Internal server error"
// @Router /api/login [post]
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

type jsonTokenResponse struct {
	Token string `json:"token"`
}

// @Summary Refresh access token
// @Description Refresh access token with user's refresh token
// @Produce json
// @Param Authorization header string true "Refresh token"
// @Success 200 {object} handler.jsonTokenResponse "New access token"
// @Failure 400 {object} handler.responseError "Something is wrong in provided information"
// @Failure 401 {object} handler.responseError "User is unauthorized"
// @Failure 500 {object} handler.responseError "Internal server error"
// @Router /api/refresh [post]
func (ah *ApiHandler) refreshAccessToken(rw http.ResponseWriter, req *http.Request) {
	tokenServ := service.TokenService{Queries: ah.ApiCfg.Queries}
	newToken, status, err := tokenServ.RefreshAccessToken(req.Context(), req.Header, ah.ApiCfg)
	if err != nil {
		respondWithError(rw, status, fmt.Sprintf("error in token refreshing: %s", err))
		return
	}
	jsonToken := jsonTokenResponse{Token: newToken}
	respondWithJson(rw, status, jsonToken)
}

// @Summary Revoke refresh token
// @Description Revoke specific refresh token
// @Param Authorization header string true "Refresh token"
// @Success 204
// @Failure 400 {object} handler.responseError "Something is wrong in provided information"
// @Router /api/revoke [post]
func (ah *ApiHandler) revokeRefreshToken(rw http.ResponseWriter, req *http.Request) {
	tokenServ := service.TokenService{Queries: ah.ApiCfg.Queries}
	status, err := tokenServ.RevokeRefreshToken(req.Context(), req.Header)
	if err != nil {
		respondWithError(rw, status, fmt.Sprintf("cannot revoke token: %s", err))
		return
	}
	respondWithJson(rw, status, nil)
}

// @Summary Update user's credentials
// @Description Update specific user's credentials by it's access token
// @Produce json
// @Param Authorization header string true "Access token"
// @Param email body string true "User's new email"
// @Param password body string true "User's new password"
// @Success 200 {object} models.UserResponse "User with updated credentials"
// @Failure 400 {object} handler.responseError "Something is wrong in provided information"
// @Failure 401 {object} handler.responseError "User is unauthorized"
// @Failure 500 {object} handler.responseError "Internal server error"
// @Router /api/users [put]
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

// @Summary Delete message
// @Description Delete specific message by it's id
// @Param messageID path string true "ID of message that needs to be deleted"
// @Success 204
// @Failure 400 {object} handler.responseError "Something is wrong in provided information"
// @Failure 401 {object} handler.responseError "User is unauthorized"
// @Failure 403 {object} handler.responseError "User cannot delete this message"
// @Failure 404 {object} handler.responseError "Message is not found"
// @Failure 500 {object} handler.responseError "Internal server error"
// @Router /api/messages/{messageID} [delete]
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

// @Summary Delete message
// @Description Delete specific message by it's id
// @Param messageID path string true "ID of message that needs to be deleted"
// @Success 204
// @Failure 401 {object} handler.responseError "Wrong api key"
// @Failure 500 {object} handler.responseError "Internal server error"
// @Router /api/payment/webhook [post]
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
}
