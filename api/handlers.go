package api

import (
	"encoding/json"
	"net/http"

	"github.com/unkeep/alfabooker/budget"
)

type handler struct {
	authToken    string
	budgetDomain *budget.Domain
}

func (h *handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// health check
	if request.URL.Path == "/" {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("OK"))
		return
	}

	token := request.Header.Get("Auth-Token")
	if token != h.authToken {
		writer.WriteHeader(http.StatusUnauthorized)
		return
	}

	if request.Method == "GET" && request.URL.Path == "/budget_stat" {
		// TODO: add auth
		h.showBudgetStat(request, writer)
		return
	}

	if request.Method == "POST" && request.URL.Path == "/account" {
		// TODO: add auth
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
		Sms string `json:"sms"`
	}

	if err := json.NewDecoder(request.Body).Decode(&reqData); err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte(err.Error()))
		return
	}

	err := h.budgetDomain.UpdateAccountBalanceFromSMS(request.Context(), reqData.Sms)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte(err.Error()))
		return
	}

	writer.WriteHeader(http.StatusOK)
}
