package service

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/ech00wv/SNserver/internal/auth"
	"github.com/ech00wv/SNserver/internal/config"
	"github.com/ech00wv/SNserver/internal/database"
	"github.com/ech00wv/SNserver/internal/models"
	"github.com/google/uuid"
)

type MessageService struct {
	ApiConfig *config.ApiConfig
}

func (messageServ *MessageService) GetMessage(ctx context.Context, messageId string) (models.MessageResponse, int, error) {
	if messageId == "" {
		return models.MessageResponse{}, http.StatusBadRequest, fmt.Errorf("message id not specified")
	}

	messageUuid, err := uuid.Parse(messageId)
	if err != nil {
		return models.MessageResponse{}, http.StatusInternalServerError, fmt.Errorf("error in converting message id to uuid: %s", err)
	}

	dbMessage, err := messageServ.ApiConfig.Queries.GetMessage(ctx, messageUuid)
	if err != nil {
		return models.MessageResponse{}, http.StatusNotFound, fmt.Errorf("error in getting message by id: %s", err)
	}

	responseMessage := converDbToMessage(dbMessage)
	return responseMessage, http.StatusOK, nil
}

func (messageServ *MessageService) GetAllMessages(ctx context.Context, authorID string, order string) ([]models.MessageResponse, int, error) {
	var (
		messages []database.Message
		err      error
	)

	if authorID != "" {
		var authorUUID uuid.UUID
		authorUUID, err = uuid.Parse(authorID)
		if err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("wrong author id: %s", err)
		}
		messages, err = messageServ.ApiConfig.Queries.GetAllMessagesForAuthor(ctx, authorUUID)
	} else {
		messages, err = messageServ.ApiConfig.Queries.GetAllMessages(ctx)
	}
	if err != nil {
		return nil, http.StatusNotFound, fmt.Errorf("cannot get messages: %s", err)
	}

	responseMessages := make([]models.MessageResponse, len(messages))
	for i, message := range messages {
		responseMessages[i] = converDbToMessage(message)
	}

	switch order {
	case "asc", "":
		sort.Slice(responseMessages, func(i, j int) bool {
			return responseMessages[i].CreatedAt.Before(responseMessages[j].CreatedAt)
		})
	case "desc":
		sort.Slice(responseMessages, func(i, j int) bool {
			return responseMessages[i].CreatedAt.After(responseMessages[j].CreatedAt)
		})
	default:
		return nil, http.StatusBadRequest, fmt.Errorf("wrong sorting order")
	}
	return responseMessages, http.StatusOK, nil
}

func (messageServ *MessageService) CreateMessage(ctx context.Context, header http.Header, messageStruct models.MessageRequest) (models.MessageResponse, int, error) {

	token, err := auth.GetBearerToken(header)
	if err != nil {
		return models.MessageResponse{}, http.StatusBadRequest, fmt.Errorf("error in getting token: %s", err)
	}
	userId, err := auth.ValidateJWT(token, messageServ.ApiConfig.JWTSecret)
	if err != nil {
		return models.MessageResponse{}, http.StatusUnauthorized, err
	}

	userExists, err := messageServ.ApiConfig.Queries.CheckUserExists(ctx, userId)
	if err != nil {
		return models.MessageResponse{}, http.StatusInternalServerError, fmt.Errorf("error in user validation: %s", err)
	}

	if !userExists {
		return models.MessageResponse{}, http.StatusBadRequest, fmt.Errorf("user does not exists")
	}

	messageText := messageStruct.Body

	valid := validateMessageText(&messageText)

	if !valid {
		return models.MessageResponse{}, http.StatusBadRequest, fmt.Errorf("message is not valid")
	}

	dbMessage, err := messageServ.ApiConfig.Queries.CreateMessage(ctx, database.CreateMessageParams{Body: messageText, UserID: userId})
	if err != nil {
		return models.MessageResponse{}, http.StatusInternalServerError, fmt.Errorf("cannot create message: %s", err)
	}
	responseMessage := converDbToMessage(dbMessage)
	return responseMessage, http.StatusCreated, nil
}

func (messageServ *MessageService) DeleteMessage(ctx context.Context, header http.Header, messageID string) (int, error) {
	token, err := auth.GetBearerToken(header)
	if err != nil {
		return http.StatusUnauthorized, fmt.Errorf("cannot find authentication header: %s", err)
	}

	userID, err := auth.ValidateJWT(token, messageServ.ApiConfig.JWTSecret)
	if err != nil {
		return http.StatusUnauthorized, fmt.Errorf("cannot validate JWT: %s", err)
	}

	messageUUID, err := uuid.Parse(messageID)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("cannot convert message id to uuid: %s", err)
	}

	_, err = messageServ.ApiConfig.Queries.GetMessage(ctx, messageUUID)
	if err != nil {
		return http.StatusNotFound, fmt.Errorf("this message does not exist: %s", err)
	}

	dbMessageID, err := messageServ.ApiConfig.Queries.DeleteMessage(ctx, database.DeleteMessageParams{ID: messageUUID, UserID: userID})
	if err != nil || dbMessageID != messageUUID {
		return http.StatusForbidden, fmt.Errorf("user cannot delete this message")
	}

	return http.StatusNoContent, nil
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

func converDbToMessage(dbMessage database.Message) models.MessageResponse {
	return models.MessageResponse{
		ID:        dbMessage.ID,
		CreatedAt: dbMessage.CreatedAt,
		UpdatedAt: dbMessage.UpdatedAt,
		Body:      dbMessage.Body,
		UserID:    dbMessage.UserID,
	}
}
