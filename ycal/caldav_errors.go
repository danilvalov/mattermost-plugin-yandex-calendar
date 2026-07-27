package ycal

import (
	"fmt"
	"net/http"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
)

func wrapCalDAVWriteErr(op string, err error) error {
	if err == nil {
		return nil
	}
	if code := remote.HTTPStatus(err); code != 0 {
		return remote.NewStatusError(code, caldavWriteMessage(op, code), err)
	}
	return err
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
