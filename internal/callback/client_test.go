package callback

import (
	"testing"

	"xendit-api-mock/internal/domain"
)

func TestSendMissingURL(t *testing.T) {
	client := NewClient("", "", nil)
	if err := client.Send(domain.CallbackPayload{ExternalID: "ext"}); err == nil {
		t.Fatal("expected error when CALLBACK_URL is missing")
	}
}
