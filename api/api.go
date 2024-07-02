package api

import (
	//"encoding/json"
	//"fmt"

	"context"
	"encoding/json"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
	"sigs.k8s.io/external-dns/provider"
)

const (
	MediaTypeFormatAndVersion = "application/external.dns.webhook+json;version=1"
	ContentTypeHeader         = "Content-Type"
)

type ApiServer struct {
	Provider provider.Provider
}

// StartHTTPApi starts a HTTP server given MiDaas provider.
// The server will listen on port `providerPort`.
// The server will respond to the following endpoints:
// - / (GET): initialization, negotiates headers and returns the domain filter
// - /records (GET): returns the current records
// - /records (POST): applies the changes
// - /adjustendpoints (POST): executes the AdjustEndpoints method
func StartHTTPApi(provider provider.Provider, readTimeout, writeTimeout time.Duration, providerPort string) {
	p := ApiServer{
		Provider: provider,
	}

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	m := http.NewServeMux()
	m.HandleFunc("/healtz", p.GetHealthz)
	m.HandleFunc("/", p.NegotiateHandler)
	m.HandleFunc("/records", p.RecordsHandler)
	m.HandleFunc("/adjustendpoints", p.AdjustEndpointsHandler)

	s := &http.Server{
		Addr:         providerPort,
		Handler:      m,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}
	log.Println("Listening on port", providerPort)
	err := s.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func (p *ApiServer) NegotiateHandler(w http.ResponseWriter, req *http.Request) {
	log.WithFields(log.Fields{"remoteAddr": req.RemoteAddr, "method": req.Method}).Info("")
	log.WithFields(log.Fields{"header": ContentTypeHeader, "version": MediaTypeFormatAndVersion}).Info("negociate header")
	w.Header().Set(ContentTypeHeader, MediaTypeFormatAndVersion)
	json.NewEncoder(w).Encode(p.Provider.GetDomainFilter())
}

func (p *ApiServer) RecordsHandler(w http.ResponseWriter, req *http.Request) {
	log.WithFields(log.Fields{"remoteAddr": req.RemoteAddr, "method": req.Method}).Info("")
	switch req.Method {
	case http.MethodGet:
		records, err := p.Provider.Records(context.Background())
		if err != nil {
			log.Errorf("Failed to get Records: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set(ContentTypeHeader, MediaTypeFormatAndVersion)
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(records); err != nil {
			log.Errorf("Failed to encode records: %v", err)
		}
		return
	case http.MethodPost:
		var changes plan.Changes
		if err := json.NewDecoder(req.Body).Decode(&changes); err != nil {
			log.Errorf("Failed to decode changes: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		err := p.Provider.ApplyChanges(context.Background(), &changes)
		if err != nil {
			log.Errorf("Failed to apply changes: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	default:
		log.Errorf("Unsupported method %s", req.Method)
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (p *ApiServer) AdjustEndpointsHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		log.Errorf("Unsupported method %s", req.Method)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	pve := []*endpoint.Endpoint{}
	if err := json.NewDecoder(req.Body).Decode(&pve); err != nil {
		log.Errorf("Failed to decode in adjustEndpointsHandler: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set(ContentTypeHeader, MediaTypeFormatAndVersion)
	pve, err := p.Provider.AdjustEndpoints(pve)
	if err != nil {
		log.Errorf("Failed to call adjust endpoints: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
	if err := json.NewEncoder(w).Encode(&pve); err != nil {
		log.Errorf("Failed to encode in adjustEndpointsHandler: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (p *ApiServer) GetHealthz(w http.ResponseWriter, req *http.Request) {
	w.Header().Set(ContentTypeHeader, MediaTypeFormatAndVersion)
}
