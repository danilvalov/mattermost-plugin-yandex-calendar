// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package remote

import (
	"fmt"
	"net/http"
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
