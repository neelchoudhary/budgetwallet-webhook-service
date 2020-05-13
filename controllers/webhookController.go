package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/neelchoudhary/budgetwallet-api-server/models"

	"github.com/neelchoudhary/budgetwallet-api-server/services/plaidfinances"
	log "github.com/sirupsen/logrus"
)

var logger = func(methodName string, err error) *log.Entry {
	if err != nil {
		return log.WithFields(log.Fields{"service": "WebhookController", "method": methodName, "error": err.Error(),
			"date": time.Now().Local().Format("01/02/2006 03:04:05")})
	}
	return log.WithFields(log.Fields{"service": "WebhookController", "method": methodName,
		"date": time.Now().Local().Format("01/02/2006 03:04:05")})
}

// WebhookController ...
type WebhookController struct {
	plaidFinancesServiceClient plaidfinances.PlaidFinancesServiceClient
}

// NewWebhookController return new controller
func NewWebhookController(plaidFinancesServiceClient plaidfinances.PlaidFinancesServiceClient) *WebhookController {
	return &WebhookController{
		plaidFinancesServiceClient: plaidFinancesServiceClient,
	}
}

// ReceiveWebhook receive webhooks from Plaid
func (c *WebhookController) ReceiveWebhook(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	userIDString := params["user_id"]
	userID, err := strconv.ParseInt(userIDString, 10, 64)
	if err != nil {
		logger("ReceiveWebhook", err).Error(fmt.Sprintf("Unable to retrieve user id"))
		http.Error(w, http.StatusText(500), http.StatusInternalServerError)
		return
	}
	var webhook models.PlaidWebhook
	err = json.NewDecoder(r.Body).Decode(&webhook)
	if err != nil {
		logger("ReceiveWebhook", err).Error(fmt.Sprintf("Unable to decode webhook"))
		http.Error(w, http.StatusText(500), http.StatusInternalServerError)
		return
	}
	logger("ReceiveWebhook", err).Info(fmt.Sprintf("Incoming Webhook - Code: %s, Type: %s, New Count: %d, Remove Count: %d.", webhook.WebhookCode, webhook.WebhookType, webhook.NewTransactionCount, len(webhook.RemovedTransactions)))
	if webhook.WebhookCode == "HISTORICAL_UPDATE" {
		req := &plaidfinances.AddHistoricalFinancialTransactionsRequest{
			UserId:      userID,
			PlaidItemId: webhook.ItemIDPlaid,
		}
		res, err := c.plaidFinancesServiceClient.AddHistoricalFinancialTransactions(context.Background(), req)
		if err != nil {
			logger("ReceiveWebhook", err).Error(fmt.Sprintf("Service call to AddHistoricalFinancialTransactions failed"))
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			return
		}
		if res.GetNewTransactions() != int64(webhook.NewTransactionCount) {
			logger("ReceiveWebhook", err).Warn(fmt.Sprintf("Webhook Mismatch - Code: %s, Type: %s, Expected Count: %d, Actual Count: %d.", webhook.WebhookCode, webhook.WebhookType, webhook.NewTransactionCount, res.GetNewTransactions()))
		} else {
			logger("ReceiveWebhook", err).Info(fmt.Sprintf("Webhook Success - Code: %s, Type: %s, New Count: %d", webhook.WebhookCode, webhook.WebhookType, res.GetNewTransactions()))
		}
	} else if webhook.WebhookCode == "DEFAULT_UPDATE" {
		req := &plaidfinances.AddFinancialTransactionsRequest{
			UserId:      userID,
			PlaidItemId: webhook.ItemIDPlaid,
		}
		res, err := c.plaidFinancesServiceClient.AddFinancialTransactions(context.Background(), req)
		if err != nil {
			logger("ReceiveWebhook", err).Error(fmt.Sprintf("Service call to AddFinancialTransactions failed"))
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			return
		}
		if res.GetNewTransactions() != int64(webhook.NewTransactionCount) {
			logger("ReceiveWebhook", err).Warn(fmt.Sprintf("Webhook Mismatch - Code: %s, Type: %s, Expected Count: %d, Actual Count: %d.", webhook.WebhookCode, webhook.WebhookType, webhook.NewTransactionCount, res.GetNewTransactions()))
		} else {
			logger("ReceiveWebhook", err).Info(fmt.Sprintf("Webhook Success - Code: %s, Type: %s, New Count: %d", webhook.WebhookCode, webhook.WebhookType, res.GetNewTransactions()))
		}
	} else if webhook.WebhookCode == "TRANSACTIONS_REMOVED" {
		req := &plaidfinances.RemoveFinancialTransactionsRequest{
			UserId:              userID,
			PlaidTransactionIds: webhook.RemovedTransactions,
		}
		_, err := c.plaidFinancesServiceClient.RemoveFinancialTransactions(context.Background(), req)
		if err != nil {
			logger("ReceiveWebhook", err).Error(fmt.Sprintf("Service call to RemoveFinancialTransactions failed"))
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			return
		}
		logger("ReceiveWebhook", err).Info(fmt.Sprintf("Webhook Success - Code: %s, Type: %s, Removed Count: %d", webhook.WebhookCode, webhook.WebhookType, len(webhook.RemovedTransactions)))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
