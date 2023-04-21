package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/unkeep/alfabooker/budget"
)

var PathPrefix = "/api"

type handler struct {
	authToken    string
	budgetDomain *budget.Domain
}

func (h *handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	path := strings.TrimSuffix(request.URL.Path, PathPrefix)
	if len(path) == len(request.URL.Path) {
		writer.WriteHeader(http.StatusNotFound)
	}

	// health check
	if path == "/" || path == "" {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("OK"))
		return
	}

	token := request.Header.Get("Auth-Token")
	if token != h.authToken {
		writer.WriteHeader(http.StatusUnauthorized)
		return
	}

	if request.Method == "GET" && path == "/budget_stat" {
		h.showBudgetStat(request, writer)
		return
	}

	if request.Method == "POST" && path == "/account" {
		h.updateAccount(request, writer)
		return
	}

	writer.WriteHeader(http.StatusNotFound)
}

func (h *handler) showBudgetStat(request *http.Request, writer http.ResponseWriter) {
	stat, err := h.budgetDomain.GetStat(request.Context())
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte(err.Error()))
		// TODO: send err to tg
		return
	}

	_ = json.NewEncoder(writer).Encode(stat)
}

func (h *handler) updateAccount(request *http.Request, writer http.ResponseWriter) {
	var reqData struct {
		Sms       string      `json:"sms"`
		Timestamp interface{} `json:"timestamp"`
	}

	if err := json.NewDecoder(request.Body).Decode(&reqData); err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte(err.Error()))
		return
	}

	fmt.Println("reqData.Timestamp: ", reqData.Timestamp)

	err := h.budgetDomain.UpdateAccountBalanceFromSMS(request.Context(), reqData.Sms)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte(err.Error()))
		return
	}

	writer.WriteHeader(http.StatusOK)
}
