package polyesterexamples

import (
	"errors"
	"fmt"
	"testing"

	sdkerrors "github.com/Fabric-Labs/polyester-sdk-go/errors"
)

func TestUniqueClientOrderIDWithinLimit(t *testing.T) {
	const maxLen = 36
	for _, prefix := range []string{"", "example-limit", "rsi-bot", "very-long-example-prefix-that-exceeds-limit"} {
		id := UniqueClientOrderID(prefix)
		if len(id) == 0 || len(id) > maxLen {
			t.Fatalf("UniqueClientOrderID(%q) = %q (len=%d), want 1..%d", prefix, id, len(id), maxLen)
		}
	}
}

func TestIsNotFoundOnlyAcceptsAPINotFound(t *testing.T) {
	notFound := fmt.Errorf("cleanup: %w", &sdkerrors.APIError{Code: "not_found"})
	if !isNotFound(notFound) {
		t.Fatal("wrapped API not_found should be successful cleanup")
	}
	if isNotFound(&sdkerrors.APIError{Code: "permission_denied"}) {
		t.Fatal("other API errors must remain failures")
	}
	if isNotFound(errors.New("not_found")) {
		t.Fatal("plain error text must not be treated as API not_found")
	}
}
