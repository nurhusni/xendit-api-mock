package httptransport

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"xendit-api-mock/internal/service/disbursement"
)

type Handler struct {
	service     *disbursement.Service
	callbackURL string
}

func NewHandler(service *disbursement.Service, callbackURL string) *Handler {
	return &Handler{service: service, callbackURL: callbackURL}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("/xendit/disbursements", loggingHandler("handleCreateDisbursement", http.HandlerFunc(h.handleCreateDisbursement)))
	mux.Handle("/xendit/healthz", loggingHandler("handleHealth", http.HandlerFunc(h.handleHealth)))
	mux.Handle("/xendit/healthz-callback", loggingHandler("handleCallbackHealth", http.HandlerFunc(h.handleCallbackHealth)))
	mux.Handle("/xendit/simulate/success", loggingHandler("handleSimulateSuccess", http.HandlerFunc(h.handleSimulateSuccess)))
	mux.Handle("/xendit/reset", loggingHandler("handleReset", http.HandlerFunc(h.handleReset)))
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleCallbackHealth(w http.ResponseWriter, r *http.Request) {
	if h.callbackURL == "" {
		log.Printf("[handleCallbackHealth] CALLBACK_URL is not set")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error"})
		return
	}
	request, err := http.NewRequest(http.MethodPost, h.callbackURL, nil)
	if err != nil {
		log.Printf("[handleCallbackHealth] request build failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error"})
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(request)
	if err != nil {
		log.Printf("[handleCallbackHealth] request failed: %v", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[handleCallbackHealth] response read failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error"})
		return
	}

	if len(body) == 0 {
		log.Printf("[handleCallbackHealth] response status=%d body={empty}", resp.StatusCode)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	if json.Valid(body) {
		var payload interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("[handleCallbackHealth] response status=%d body=%s", resp.StatusCode, string(body))
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		}
		pretty, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			log.Printf("[handleCallbackHealth] response status=%d body=%s", resp.StatusCode, string(body))
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		}
		log.Printf("[handleCallbackHealth] response status=%d json=%s", resp.StatusCode, string(pretty))
	} else {
		log.Printf("[handleCallbackHealth] response status=%d body=%s", resp.StatusCode, string(body))
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleCreateDisbursement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	req, err := decodeDisbursementRequest(r)
	if err != nil {
		log.Printf("[handleCreateDisbursement] decode failed: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	resp, cbErr := h.service.Create(req)
	if cbErr != nil {
		log.Printf("[handleCreateDisbursement] callback failed: %v", cbErr)
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleSimulateSuccess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	req, err := decodeDisbursementRequest(r)
	if err != nil {
		log.Printf("[handleSimulateSuccess] decode failed: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	resp, cbErr := h.service.SimulateSuccess(req)
	if cbErr != nil {
		log.Printf("[handleSimulateSuccess] callback failed: %v", cbErr)
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	h.service.Reset()
	writeJSON(w, http.StatusOK, map[string]string{"status": "reset"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
