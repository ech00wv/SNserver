package service

import (
	"context"
	"net/http"

	"github.com/ech00wv/SNserver/internal/auth"
	"github.com/ech00wv/SNserver/internal/config"
	"github.com/ech00wv/SNserver/internal/models"
	"github.com/google/uuid"
)

type PaymentService struct {
	ApiConfig *config.ApiConfig
}

func (paymentServ *PaymentService) UpgradeToPremium(ctx context.Context, paymentData models.PaymentProviderWebhook, reqHeader http.Header) int {

	token, err := auth.GetApiKey(reqHeader)
	if err != nil {
		return http.StatusUnauthorized
	}

	if token != paymentServ.ApiConfig.PaymentKey {
		return http.StatusUnauthorized
	}

	if paymentData.Event != "user.upgraded" {
		return http.StatusNoContent
	}

	userID := paymentData.Data.UserID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return http.StatusNotFound
	}

	err = paymentServ.ApiConfig.Queries.UpgradeToPremium(ctx, userUUID)
	if err != nil {
		return http.StatusInternalServerError
	}

	return http.StatusNoContent
}
