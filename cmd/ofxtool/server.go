package main

import (
	"net/http"

	"github.com/ghetzel/diecast"
	"github.com/ghetzel/go-stockutil/httputil"
)

type Server struct {
	client *Client
	dc     *diecast.Server
}

func NewServer(client *Client) *Server {
	return &Server{
		client: client,
		dc:     diecast.NewServer(`ui`),
	}
}

func (self *Server) ListenAndServe(address string) error {
	self.dc.Get(`/ofxtool/v1/`, func(w http.ResponseWriter, req *http.Request) {
		httputil.RespondJSON(w, map[string]interface{}{
			`status`:  `ok`,
			`address`: address,
		})
	})

	// Transactions
	// ---------------------------------------------------------------------------------------------
	self.dc.Get(`/ofxtool/v1/transactions`, func(w http.ResponseWriter, req *http.Request) {
		httputil.RespondJSON(w, nil, http.StatusNotExtended)
	})

	self.dc.Get(`/ofxtool/v1/transactions/:id`, func(w http.ResponseWriter, req *http.Request) {
		httputil.RespondJSON(w, nil, http.StatusNotExtended)
	})

	// Accounts
	// ---------------------------------------------------------------------------------------------
	self.dc.Get(`/ofxtool/v1/accounts/`, func(w http.ResponseWriter, req *http.Request) {
		httputil.RespondJSON(w, nil, http.StatusNotExtended)
	})

	self.dc.Get(`/ofxtool/v1/accounts/:institution/:id`, func(w http.ResponseWriter, req *http.Request) {
		httputil.RespondJSON(w, nil, http.StatusNotExtended)
	})

	self.dc.Post(`/ofxtool/v1/accounts/`, func(w http.ResponseWriter, req *http.Request) {
		httputil.RespondJSON(w, nil, http.StatusNotExtended)
	})

	self.dc.Post(`/ofxtool/v1/accounts/:institution/:id`, func(w http.ResponseWriter, req *http.Request) {
		httputil.RespondJSON(w, nil, http.StatusNotExtended)
	})

	// Institutions
	// ---------------------------------------------------------------------------------------------
	self.dc.Get(`/ofxtool/v1/institutions`, func(w http.ResponseWriter, req *http.Request) {
		httputil.RespondJSON(w, nil, http.StatusNotExtended)
	})

	self.dc.Get(`/ofxtool/v1/institutions/:id`, func(w http.ResponseWriter, req *http.Request) {
		httputil.RespondJSON(w, nil, http.StatusNotExtended)
	})

	self.dc.Post(`/ofxtool/v1/institutions/:id`, func(w http.ResponseWriter, req *http.Request) {
		httputil.RespondJSON(w, nil, http.StatusNotExtended)
	})

	// Payees
	// ---------------------------------------------------------------------------------------------
	self.dc.Get(`/ofxtool/v1/payees`, func(w http.ResponseWriter, req *http.Request) {
		httputil.RespondJSON(w, nil, http.StatusNotExtended)
	})

	self.dc.Get(`/ofxtool/v1/payees/:id`, func(w http.ResponseWriter, req *http.Request) {
		httputil.RespondJSON(w, nil, http.StatusNotExtended)
	})

	self.dc.Post(`/ofxtool/v1/payees/:id`, func(w http.ResponseWriter, req *http.Request) {
		httputil.RespondJSON(w, nil, http.StatusNotExtended)
	})

	return self.dc.ListenAndServe(address)
}
