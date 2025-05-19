package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/unkeep/alfabooker/budget"
)

var PathPrefix = "/api"

type handler struct {
	authToken    string
	budgetDomain *budget.Domain
}

func (h *handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	path := strings.TrimPrefix(request.URL.Path, PathPrefix)
	if len(path) == len(request.URL.Path) {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	// health check
	if path == "/" || path == "" {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("OK"))
		return
	}

	if request.Method == "GET" && path == "/progress_csv" {
		h.progressCSV(request, writer)
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

func (h *handler) progressCSV(request *http.Request, writer http.ResponseWriter) {
	stat, err := h.budgetDomain.GetStat(request.Context())
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte(err.Error()))
		// TODO: send err to tg
		return
	}

	balancePct := int64(stat.TotalBalance / stat.BudgetAmount * 100.0)

	totalTime := stat.BudgetExpiresAt - stat.BudgetStartedAt
	timeElapsed := time.Now().Unix() - stat.BudgetStartedAt

	timeElapsedPct := int(float64(timeElapsed) / float64(totalTime) * 100.0)

	writer.Header().Set("Content-Type", "text/csv")
	_, _ = fmt.Fprintf(writer, "balance,elapsed\n")
	_, _ = fmt.Fprintf(writer, "%d,%d\n", balancePct, timeElapsedPct)
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
