package ycal

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
)

func TestWrapCalDAVWriteErr(t *testing.T) {
	t.Parallel()

	cause := fmt.Errorf("%d Forbidden: denied", http.StatusForbidden)
	wrapped := fmt.Errorf("ycal Put: %w", cause)
	err := wrapCalDAVWriteErr("update", wrapped)
	var se *remote.StatusError
	if !errors.As(err, &se) {
		t.Fatalf("got %T %v", err, err)
	}
	if se.Code != http.StatusForbidden {
		t.Fatalf("code=%d", se.Code)
	}
	if !strings.Contains(se.Error(), "denied") {
		t.Fatalf("message=%q", se.Error())
	}

	passthru := errors.New("network down")
	if got := wrapCalDAVWriteErr("update", passthru); !errors.Is(got, passthru) {
		t.Fatalf("got %v", got)
	}
}
