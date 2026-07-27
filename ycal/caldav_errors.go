package ycal

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
)

func wrapCalDAVWriteErr(op string, err error) error {
	if err == nil {
		return nil
	}
	if code := httpStatusFromErr(err); code != 0 {
		return remote.NewStatusError(code, caldavWriteMessage(op, code), err)
	}
	return err
}

func httpStatusFromErr(err error) int {
	for e := err; e != nil; e = errors.Unwrap(e) {
		if code := parseLeadingHTTPStatus(e.Error()); code != 0 {
			return code
		}
	}
	// pkg/errors wrap: "context: 403 Forbidden: detail"
	s := err.Error()
	if i := strings.LastIndex(s, ": "); i >= 0 {
		if code := parseLeadingHTTPStatus(s[i+2:]); code != 0 {
			return code
		}
	}
	return 0
}

func parseLeadingHTTPStatus(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	end := 0
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	if end != 3 {
		return 0
	}
	if end < len(s) && s[end] != ' ' && s[end] != ':' {
		return 0
	}
	code, err := strconv.Atoi(s[:end])
	if err != nil || code < 400 || code > 599 {
		return 0
	}
	return code
}

func caldavWriteMessage(op string, code int) string {
	switch code {
	case http.StatusForbidden:
		return fmt.Sprintf("Yandex Calendar denied the %s (no permission to edit this event)", op)
	case http.StatusConflict, http.StatusPreconditionFailed:
		return fmt.Sprintf("Yandex Calendar conflict while saving the %s", op)
	case http.StatusNotFound:
		return fmt.Sprintf("Yandex Calendar %s failed: event not found", op)
	default:
		if code >= 400 && code < 500 {
			return fmt.Sprintf("Yandex Calendar rejected the %s (%d %s)", op, code, http.StatusText(code))
		}
		return fmt.Sprintf("Yandex Calendar %s failed (%d %s)", op, code, http.StatusText(code))
	}
}
