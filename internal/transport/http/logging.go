package httptransport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type responseRecorder struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	r.body.Write(data)
	return r.ResponseWriter.Write(data)
}

func loggingHandler(name string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("[%s.logRequest] request read failed method=%s path=%s error=%v", name, r.Method, r.URL.Path, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_ = r.Body.Close()
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		logRequest(name, r, bodyBytes)

		recorder := &responseRecorder{ResponseWriter: w}
		defer func() {
			if recErr := recover(); recErr != nil {
				log.Printf("[%s.logResponse] request panic method=%s path=%s error=%v", name, r.Method, r.URL.Path, recErr)
				recorder.WriteHeader(http.StatusInternalServerError)
			}
			logResponse(name, r, recorder)
		}()

		next.ServeHTTP(recorder, r)
	})
}

func logRequest(name string, r *http.Request, body []byte) {
	bodyLog := formatBody(body)
	log.Printf("[%s.logRequest] request method=%s path=%s body=%s", name, r.Method, r.URL.Path, bodyLog)
}

func logResponse(name string, r *http.Request, recorder *responseRecorder) {
	status := recorder.status
	if status == 0 {
		status = http.StatusOK
	}
	responseBody := formatBody(recorder.body.Bytes())
	if status >= http.StatusBadRequest {
		log.Printf("[%s.logResponse] response error method=%s path=%s status=%d body=%s", name, r.Method, r.URL.Path, status, responseBody)
		return
	}
	log.Printf("[%s.logResponse] response success method=%s path=%s status=%d body=%s", name, r.Method, r.URL.Path, status, responseBody)
}

func formatBody(body []byte) string {
	if len(body) == 0 {
		return "{empty}"
	}
	if json.Valid(body) {
		var payload interface{}
		if err := json.Unmarshal(body, &payload); err == nil {
			pretty, err := json.MarshalIndent(payload, "", "  ")
			if err == nil {
				return fmt.Sprintf("%s", string(pretty))
			}
		}
	}
	return string(body)
}
