// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Package caldavconnect registers HTTP routes for app-password (CalDAV) account linking.
package caldavconnect

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html"
	"net/http"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/config"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/httputils"
)

const (
	caldavConnectTokenTTLSeconds = int64(15 * 60)
	caldavConnectTokenHexLen     = 64 // hex-encoded 32 random bytes
)

// PluginKV is optional plugin KV used for one-time connect tokens. Some Mattermost
// versions or clients omit Mattermost-User-ID on POST to plugin routes; a short-lived
// token minted on GET avoids relying on that header for the form submit.
type PluginKV interface {
	KVSetWithExpiry(key string, value []byte, expireInSeconds int64) *model.AppError
	KVGet(key string) ([]byte, *model.AppError)
	KVDelete(key string) *model.AppError
}

// App completes CalDAV / app-password authentication.
type App interface {
	CompleteCalDAVConnect(mattermostUserID, email, appPassword string) error
}

type handler struct {
	app      App
	provider config.ProviderConfig
	kv       PluginKV
}

// Init registers /caldav/connect (GET shows form, POST submits credentials).
// If kv is non-nil, GET mints a one-time token stored in plugin KV so POST can
// identify the Mattermost user when Mattermost-User-ID is missing on submit.
func Init(h *httputils.Handler, app App, provider config.ProviderConfig, kv PluginKV) {
	o := &handler{app: app, provider: provider, kv: kv}
	r := h.Router.PathPrefix("/caldav").Subrouter()
	r.HandleFunc("/connect", o.connectGET).Methods(http.MethodGet)
	r.HandleFunc("/connect", o.connectPOST).Methods(http.MethodPost)
}

func (o *handler) connectGET(w http.ResponseWriter, r *http.Request) {
	mattermostUserID := r.Header.Get("Mattermost-User-ID")
	if mattermostUserID == "" {
		http.Error(w, "Not authorized — open this URL while logged into Mattermost in your browser.", http.StatusUnauthorized)
		return
	}

	hidden := ""
	if o.kv != nil {
		tok, mintErr := o.mintConnectToken(mattermostUserID)
		if mintErr != nil {
			http.Error(w, "could not start connect session — try again", http.StatusInternalServerError)
			return
		}
		hidden = fmt.Sprintf(`<input type="hidden" name="caldav_connect_token" value="%s">`, html.EscapeString(tok))
	}

	page := fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>Connect %s</title></head>
<body>
<h1>Connect %s</h1>
<p>Use an <strong>app password</strong> from your <a href="https://yandex.ru/security/app-passwords" target="_blank">account security settings</a> (not your regular login password).</p>
<form method="POST" action="">
%s
<label>Email<br><input type="email" name="email" required size="40" autocomplete="username"></label><br><br>
<label>App password<br><input type="password" name="app_password" required size="40" autocomplete="current-password"></label><br><br>
<button type="submit">Connect</button>
</form>
</body></html>`,
		html.EscapeString(o.provider.DisplayName),
		html.EscapeString(o.provider.DisplayName),
		hidden,
	)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(page))
}

func (o *handler) connectPOST(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	mattermostUserID := r.Header.Get("Mattermost-User-ID")
	var connectToken string
	if mattermostUserID == "" && o.kv != nil {
		if tok := r.Form.Get("caldav_connect_token"); tok != "" {
			if uid, ok := o.lookupConnectToken(tok); ok {
				mattermostUserID = uid
				connectToken = tok
			}
		}
	}
	if mattermostUserID == "" {
		http.Error(w, "Not authorized — open the connect page again from Mattermost (session was not sent with this submit).", http.StatusUnauthorized)
		return
	}
	email := r.Form.Get("email")
	appPassword := r.Form.Get("app_password")
	if email == "" || appPassword == "" {
		http.Error(w, "email and app_password are required", http.StatusBadRequest)
		return
	}

	err := o.app.CompleteCalDAVConnect(mattermostUserID, email, appPassword)
	if err != nil {
		httputils.WriteUnauthorizedError(w, err)
		return
	}
	if connectToken != "" && o.kv != nil {
		_ = o.kv.KVDelete(connectTokenKVKey(connectToken))
	}

	success := fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>Connected</title></head>
<body><p>Connected to %s. You can close this window and return to Mattermost.</p>
<script>window.close();</script>
</body></html>`, html.EscapeString(o.provider.DisplayName))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(success))
}

func connectTokenKVKey(token string) string {
	return "caldav_connect_" + token
}

func (o *handler) mintConnectToken(mattermostUserID string) (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	token := hex.EncodeToString(raw[:])
	key := connectTokenKVKey(token)
	if appErr := o.kv.KVSetWithExpiry(key, []byte(mattermostUserID), caldavConnectTokenTTLSeconds); appErr != nil {
		return "", appErr
	}
	return token, nil
}

func validConnectTokenHex(tok string) bool {
	if len(tok) != caldavConnectTokenHexLen {
		return false
	}
	_, err := hex.DecodeString(tok)
	return err == nil
}

// lookupConnectToken resolves a token minted on GET. The KV entry is removed only
// after CompleteCalDAVConnect succeeds so a wrong app password can be corrected
// without reloading the page.
func (o *handler) lookupConnectToken(token string) (mattermostUserID string, ok bool) {
	if o.kv == nil || !validConnectTokenHex(token) {
		return "", false
	}
	data, appErr := o.kv.KVGet(connectTokenKVKey(token))
	if appErr != nil || len(data) == 0 {
		return "", false
	}
	return string(data), true
}
