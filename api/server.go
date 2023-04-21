package api

import (
	"crypto/tls"
	"net/http"

	"github.com/unkeep/alfabooker/budget"
)

func NewServer(port string, budgetDomain *budget.Domain, authToken string) http.Server {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	h := &handler{budgetDomain: budgetDomain, authToken: authToken}

	return http.Server{
		Addr:    "0.0.0.0:" + port,
		Handler: h,
	}
}

func NewHandler(budgetDomain *budget.Domain, authToken string) http.Handler {
	return &handler{budgetDomain: budgetDomain, authToken: authToken}
}
