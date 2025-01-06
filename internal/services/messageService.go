package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/ech00wv/SNserver/internal/auth"
	"github.com/ech00wv/SNserver/internal/config"
	"github.com/ech00wv/SNserver/internal/database"
	"github.com/ech00wv/SNserver/internal/models"
	"github.com/google/uuid"
)

type MessageService struct {
	Queries *database.Queries
}

func (messageServ *MessageService) GetMessage(ctx context.Context, messageId string) (models.Message, int, error) {
	if messageId == "" {
		return models.Message{}, http.StatusBadRequest, fmt.Errorf("message id not specified")
	}

	messageUuid, err := uuid.Parse(messageId)
	if err != nil {
		return models.Message{}, http.StatusInternalServerError, fmt.Errorf("error in converting message id to uuid: %s", err)
	}

	dbMessage, err := messageServ.Queries.GetMessage(ctx, messageUuid)
	if err != nil {
		return models.Message{}, http.StatusBadRequest, fmt.Errorf("error in getting message by id: %s", err)
	}

	responseMessage := converDbToMessage(dbMessage)
	return responseMessage, http.StatusOK, nil
}

func (messageServ *MessageService) GetAllMessages(ctx context.Context) ([]models.Message, int, error) {
	messages, err := messageServ.Queries.GetAllMessages(ctx)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("cannot get messages: %s", err)
	}
	responseMessages := make([]models.Message, len(messages))
	for i, message := range messages {
		responseMessages[i] = converDbToMessage(message)
	}
	return responseMessages, http.StatusOK, nil
}

func (messageServ *MessageService) CreateMessage(ctx context.Context, header http.Header, messageStruct models.MessageRequest, apiCfg *config.ApiConfig) (models.Message, int, error) {

	token, err := auth.GetBearerToken(header)
	if err != nil {
		return models.Message{}, http.StatusBadRequest, fmt.Errorf("error in getting token: %s", err)
	}
	userId, err := auth.ValidateJWT(token, apiCfg.JWTSecret)
	if err != nil {
		return models.Message{}, http.StatusUnauthorized, err
	}

	userExists, err := messageServ.Queries.CheckUserExists(ctx, userId)
	if err != nil {
		return models.Message{}, http.StatusInternalServerError, fmt.Errorf("error in user validation: %s", err)
	}

	if !userExists {
		return models.Message{}, http.StatusBadRequest, fmt.Errorf("user does not exists")
	}

	messageText := messageStruct.Body

	valid := validateMessageText(&messageText)

	if !valid {
		return models.Message{}, http.StatusBadRequest, fmt.Errorf("message is not valid")
	}

	dbMessage, err := messageServ.Queries.CreateMessage(ctx, database.CreateMessageParams{Body: messageText, UserID: userId})
	if err != nil {
		return models.Message{}, http.StatusInternalServerError, fmt.Errorf("cannot create message: %s", err)
	}
	responseMessage := converDbToMessage(dbMessage)
	return responseMessage, http.StatusCreated, nil
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

func converDbToMessage(dbMessage database.Message) models.Message {
	return models.Message{
		ID:        dbMessage.ID,
		CreatedAt: dbMessage.CreatedAt,
		UpdatedAt: dbMessage.UpdatedAt,
		Body:      dbMessage.Body,
		UserID:    dbMessage.UserID,
	}
}
