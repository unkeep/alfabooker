package api

import (
	"encoding/json"
	"net/http"

	"github.com/unkeep/alfabooker/budget"
)

type handler struct {
	budgetDomain *budget.Domain
}

func (h *handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method == "GET" && request.URL.Path == "/budget_stat" {
		// TODO: add auth
		h.showBudgetStat(request, writer)
		return
	}

	// health check
	if request.URL.Path == "/" {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("OK"))
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
