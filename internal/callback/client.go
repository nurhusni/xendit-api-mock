package callback

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"xendit-api-mock/internal/domain"
)

type Client struct {
	callbackURL string
	token       string
	httpClient  *http.Client
}

func NewClient(callbackURL, token string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{callbackURL: callbackURL, token: token, httpClient: httpClient}
}

func (c *Client) Send(payload domain.CallbackPayload) error {
	if c.callbackURL == "" {
		return fmt.Errorf("CALLBACK_URL is not set")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPost, c.callbackURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		request.Header.Set("X-Callback-Token", c.token)
	} else {
		log.Printf("[callback.Send] CALLBACK_TOKEN is not set")
	}
	log.Printf("[callback.Send] request method=%s url=%s body=%s", request.Method, request.URL.String(), formatBody(body))

	resp, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[callback.Send] response read failed status=%d error=%v", resp.StatusCode, err)
		return nil
	}

	log.Printf("[callback.Send] response method=%s url=%s status=%d body=%s", request.Method, request.URL.String(), resp.StatusCode, formatBody(respBody))

	return nil
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
