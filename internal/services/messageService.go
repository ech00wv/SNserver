package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/ech00wv/SNserver/internal/database"
	"github.com/ech00wv/SNserver/internal/models"
	"github.com/google/uuid"
)

type MessageService struct {
	Queries *database.Queries
}

func (messageServ *MessageService) GetMessage(ctx context.Context, messageId string) (database.Message, int, error) {
	if messageId == "" {
		return database.Message{}, http.StatusBadRequest, fmt.Errorf("message id not specified")
	}

	messageUuid, err := uuid.Parse(messageId)
	if err != nil {
		return database.Message{}, http.StatusInternalServerError, fmt.Errorf("error in converting message id to uuid")
	}

	dbMessage, err := messageServ.Queries.GetMessage(ctx, messageUuid)
	if err != nil {
		return database.Message{}, http.StatusBadRequest, fmt.Errorf("error in getting message by id")
	}

	return dbMessage, http.StatusOK, nil
}

func (messageServ *MessageService) GetAllMessages(ctx context.Context) ([]database.Message, int, error) {
	messages, err := messageServ.Queries.GetAllMessages(ctx)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("cannot get messages")
	}

	return messages, http.StatusOK, nil
}

func (messageServ *MessageService) CreateMessage(ctx context.Context, messageStruct models.MessageRequest) (database.Message, int, error) {

	userExists, err := messageServ.Queries.CheckUserExists(ctx, messageStruct.UserId)
	if err != nil {
		return database.Message{}, http.StatusInternalServerError, fmt.Errorf("error in user validation")
	}

	if !userExists {
		return database.Message{}, http.StatusBadRequest, fmt.Errorf("user does not exists")
	}

	err = validateMessageRequest(&messageStruct)
	if err != nil {
		return database.Message{}, http.StatusBadRequest, fmt.Errorf("mismatch request body")
	}

	messageText := messageStruct.Body

	valid := validateMessageText(&messageText)

	if !valid {
		return database.Message{}, http.StatusBadRequest, fmt.Errorf("message is not valid")
	}

	dbMessage, err := messageServ.Queries.CreateMessage(ctx, database.CreateMessageParams{Body: messageText, UserID: messageStruct.UserId})
	if err != nil {
		return database.Message{}, http.StatusInternalServerError, fmt.Errorf("cannot create message")
	}

	return dbMessage, http.StatusCreated, nil
}

func validateMessageRequest(mr *models.MessageRequest) error {
	if mr.Body == "" {
		return fmt.Errorf("message text is empty")
	}
	if mr.UserId == uuid.Nil {
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
