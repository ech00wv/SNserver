package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync/atomic"

	"github.com/ech00wv/SNserver/internal/database"
	"github.com/google/uuid"
)

// if something goes wrong, send error json
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
func (ac *apiConfig) middlewareMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		ac.fileserverHits.Add(1)
		next.ServeHTTP(rw, req)
	})
}

// handler to see how many requests was sent on GET /app URL by GET /admin/metrics
func (ac *apiConfig) serveMetrics(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	rw.WriteHeader(200)
	fmt.Fprintf(rw, `<html>
					<body>
						<h1>Welcome, Admin</h1>
						<p>Social Network has been visited %d times!</p>
					</body>
				</html>`, ac.fileserverHits.Load())
}

// reset whole app on POST /admin/reset
func (ac *apiConfig) resetApp(rw http.ResponseWriter, req *http.Request) {
	if ac.platfrom != "dev" {
		rw.WriteHeader(http.StatusForbidden)
		return
	}

	err := ac.queries.DeleteUsers(req.Context())
	if err != nil {
		respondWithError(rw, http.StatusInternalServerError, fmt.Sprintf("error in deleting users: %s", err))
		return
	}

	ac.fileserverHits = atomic.Int64{}
	rw.Header().Set("Content-Type", "text/plain; charset=utf8")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte("app successfuly resetted!"))
}

func emailIsCorrect(email string) bool {
	matched, _ := regexp.Match(`^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$`, []byte(email))
	return matched
}

func (ac *apiConfig) createUser(rw http.ResponseWriter, req *http.Request) {

	var reqBody struct {
		Email string `json:"email"`
	}

	err := json.NewDecoder(req.Body).Decode(&reqBody)
	defer req.Body.Close()
	if err != nil {
		respondWithError(rw, http.StatusBadRequest, fmt.Sprintf("error marshalling json: %s", err))
		return
	}

	if !emailIsCorrect(reqBody.Email) {
		respondWithError(rw, http.StatusBadRequest, "email is not valid")
		return
	}

	dbUser, err := ac.queries.CreateUser(req.Context(), reqBody.Email)
	if err != nil {
		respondWithError(rw, http.StatusInternalServerError, fmt.Sprintf("error creating user: %s", err))
		return
	}

	respondWithJson(rw, http.StatusCreated, dbUser)
}

type messageRequest struct {
	Body   string    `json:"body"`
	UserId uuid.UUID `json:"user_id"`
}

// Create message on POST /api/messages
func (ac *apiConfig) createMessage(rw http.ResponseWriter, req *http.Request) {

	var reqBody messageRequest

	err := json.NewDecoder(req.Body).Decode(&reqBody)
	defer req.Body.Close()

	if err != nil {
		respondWithError(rw, http.StatusInternalServerError, fmt.Sprintf("error decoding body: %s", err))
		return
	}

	userExists, err := ac.queries.CheckUserExists(req.Context(), reqBody.UserId)
	if err != nil {
		respondWithError(rw, http.StatusInternalServerError, fmt.Sprintf("error in user validation: %s", err))
		return
	}

	if !userExists {
		respondWithError(rw, http.StatusBadRequest, "user does not exists")
		return
	}

	err = reqBody.validateMessageRequest()
	if err != nil {
		respondWithError(rw, http.StatusBadRequest, fmt.Sprintf("mismatch request body: %s", err))
		return
	}

	messageText := reqBody.Body

	valid := validateMessageText(&messageText)

	if !valid {
		respondWithError(rw, http.StatusBadRequest, "message is not valid")
		return
	}

	dbMessage, err := ac.queries.CreateMessage(req.Context(), database.CreateMessageParams{Body: messageText, UserID: reqBody.UserId})
	if err != nil {
		respondWithError(rw, http.StatusInternalServerError, fmt.Sprintf("cannot create message: %s", err))
		return
	}

	respondWithJson(rw, http.StatusCreated, dbMessage)
}

func (cr *messageRequest) validateMessageRequest() error {
	if cr.Body == "" {
		return fmt.Errorf("message text is empty")
	}
	if cr.UserId == uuid.Nil {
		return fmt.Errorf("user id is empty")
	}
	return nil
}

func validateMessageText(message *string) bool {
	const messageMaxLength = 140
	if len(*message) > messageMaxLength {
		return false
	}
	*message = profanityFix(*message)
	return true
}

// checking profanity of sended message and censoring it
func profanityFix(msg string) string {
	profanedWords := map[string]struct{}{
		"kerfuffle": {},
		"sharbert":  {},
		"fornax":    {},
	}
	splittedString := strings.Split(msg, " ")
	splittedLoweredString := strings.Split(strings.ToLower(msg), " ")
	for i, word := range splittedLoweredString {
		if _, found := profanedWords[word]; found {
			splittedString[i] = "****"
		}
	}
	return strings.Join(splittedString, " ")
}

func (ac *apiConfig) getMessages(rw http.ResponseWriter, req *http.Request) {

	messages, err := ac.queries.GetAllMessages(req.Context())
	if err != nil {
		respondWithError(rw, http.StatusInternalServerError, fmt.Sprintf("cannot get messages: %s", err))
		return
	}

	respondWithJson(rw, http.StatusOK, messages)
}

func (ac *apiConfig) getMessage(rw http.ResponseWriter, req *http.Request) {
	messageId := req.PathValue("messageId")
	if messageId == "" {
		respondWithError(rw, http.StatusBadRequest, "message id not specified")
		return
	}

	messageUuid, err := uuid.Parse(messageId)
	if err != nil {
		respondWithError(rw, http.StatusInternalServerError, fmt.Sprintf("error in converting message id to uuid: %s", err))
		return
	}

	dbMessage, err := ac.queries.GetMessage(req.Context(), messageUuid)
	if err != nil {
		respondWithError(rw, http.StatusInternalServerError, fmt.Sprintf("error in getting message by id: %s", err))
		return
	}

	respondWithJson(rw, http.StatusOK, dbMessage)
}
