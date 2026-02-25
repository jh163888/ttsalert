package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type AlertQueue interface {
	Enqueue(alert Alert) error
}

type Alert struct {
	ID           string            `json:"id"`
	Title        string            `json:"title"`
	Message      string            `json:"message"`
	Severity     string            `json:"severity"`
	Source       string            `json:"source"`
	Host         string            `json:"host"`
	Timestamp    time.Time         `json:"timestamp"`
	PhoneNumbers []string          `json:"phone_numbers"`
	CustomData   map[string]string `json:"custom_data"`
}

type WebhookHandler struct {
	queue  AlertQueue
	logger *logrus.Logger
}

func NewWebhookHandler(queue AlertQueue, logger *logrus.Logger) *WebhookHandler {
	return &WebhookHandler{
		queue:  queue,
		logger: logger,
	}
}

func (h *WebhookHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/webhook/zabbix", h.HandleZabbix).Methods("POST")
	r.HandleFunc("/webhook/opm", h.HandleOPM).Methods("POST")
	r.HandleFunc("/webhook/generic", h.HandleGeneric).Methods("POST")
	r.HandleFunc("/health", h.HealthCheck).Methods("GET")
}

func (h *WebhookHandler) HandleZabbix(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		EventID     string `json:"eventid"`
		Title       string `json:"title"`
		Message     string `json:"message"`
		Severity    string `json:"severity"`
		Host        string `json:"host"`
		Time        string `json:"time"`
		PhoneNumber string `json:"phone_number"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	alert := Alert{
		ID:           payload.EventID,
		Title:        payload.Title,
		Message:      payload.Message,
		Severity:     payload.Severity,
		Source:       "zabbix",
		Host:         payload.Host,
		Timestamp:    time.Now(),
		PhoneNumbers: []string{payload.PhoneNumber},
	}

	if err := h.queue.Enqueue(alert); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Infof("Received Zabbix alert: %s", alert.ID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "queued"})
}

func (h *WebhookHandler) HandleOPM(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		AlertID     string            `json:"alert_id"`
		Subject     string            `json:"subject"`
		Description string            `json:"description"`
		Severity    string            `json:"severity"`
		Device      string            `json:"device"`
		Time        string            `json:"time"`
		PhoneNumber string            `json:"phone_number"`
		CustomData  map[string]string `json:"custom_data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	alert := Alert{
		ID:           payload.AlertID,
		Title:        payload.Subject,
		Message:      payload.Description,
		Severity:     payload.Severity,
		Source:       "opm",
		Host:         payload.Device,
		Timestamp:    time.Now(),
		PhoneNumbers: []string{payload.PhoneNumber},
		CustomData:   payload.CustomData,
	}

	if err := h.queue.Enqueue(alert); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Infof("Received OPM alert: %s", alert.ID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "queued"})
}

func (h *WebhookHandler) HandleGeneric(w http.ResponseWriter, r *http.Request) {
	var alert Alert
	if err := json.NewDecoder(r.Body).Decode(&alert); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if alert.ID == "" {
		alert.ID = time.Now().Format("20060102150405")
	}

	alert.Timestamp = time.Now()
	alert.Source = "generic"

	if err := h.queue.Enqueue(alert); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Infof("Received generic alert: %s", alert.ID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "queued"})
}

func (h *WebhookHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}
