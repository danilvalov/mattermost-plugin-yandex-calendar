// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package remote

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// StatusError is a CalDAV/HTTP status from the remote calendar server.
type StatusError struct {
	Code    int
	Message string
	Err     error
}

func NewStatusError(code int, message string, cause error) *StatusError {
	return &StatusError{Code: code, Message: message, Err: cause}
}

func (e *StatusError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Code > 0 {
		return fmt.Sprintf("remote calendar error (%d %s)", e.Code, http.StatusText(e.Code))
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "remote calendar error"
}

func (e *StatusError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// IsUnauthorized reports whether err represents HTTP 401 / unauthenticated from the calendar server.
func IsUnauthorized(err error) bool {
	if err == nil {
		return false
	}
	if HTTPStatus(err) == http.StatusUnauthorized {
		return true
	}
	// go-webdav FindCurrentUserPrincipal when principal is unauthenticated (no numeric status).
	return strings.Contains(strings.ToLower(err.Error()), "unauthenticated")
}

// HTTPStatus extracts an HTTP status code from err (StatusError, or text like "401 Unauthorized").
func HTTPStatus(err error) int {
	if err == nil {
		return 0
	}
	var se *StatusError
	if errors.As(err, &se) && se != nil && se.Code != 0 {
		return se.Code
	}
	for e := err; e != nil; e = errors.Unwrap(e) {
		if code := parseLeadingHTTPStatus(e.Error()); code != 0 {
			return code
		}
	}
	// pkg/errors wrap: "context: 401 Unauthorized: detail"
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
