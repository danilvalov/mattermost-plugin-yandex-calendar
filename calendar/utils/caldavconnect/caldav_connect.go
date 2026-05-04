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
	mmi18n "github.com/mattermost/mattermost/server/public/pluginapi/i18n"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/config"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/locale"
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
	i18n     *mmi18n.Bundle
}

// Init registers /caldav/connect (GET shows form, POST submits credentials).
// If kv is non-nil, GET mints a one-time token stored in plugin KV so POST can
// identify the Mattermost user when Mattermost-User-ID is missing on submit.
func Init(h *httputils.Handler, app App, provider config.ProviderConfig, kv PluginKV, bundle *mmi18n.Bundle) {
	o := &handler{app: app, provider: provider, kv: kv, i18n: bundle}
	r := h.Router.PathPrefix("/caldav").Subrouter()
	r.HandleFunc("/connect", o.connectGET).Methods(http.MethodGet)
	r.HandleFunc("/connect", o.connectPOST).Methods(http.MethodPost)
}

func (o *handler) connectGET(w http.ResponseWriter, r *http.Request) {
	mattermostUserID := r.Header.Get("Mattermost-User-ID")
	if mattermostUserID == "" {
		msg := locale.Server(o.i18n, "ycal.caldav.err.unauthorized_get",
			"Not authorized — open this URL while logged into Mattermost in your browser.", nil)
		http.Error(w, msg, http.StatusUnauthorized)
		return
	}

	hidden := ""
	if o.kv != nil {
		tok, mintErr := o.mintConnectToken(mattermostUserID)
		if mintErr != nil {
			msg := locale.User(o.i18n, mattermostUserID, "ycal.caldav.err.session_start",
				"could not start connect session — try again", nil)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}
		hidden = fmt.Sprintf(`<input type="hidden" name="caldav_connect_token" value="%s">`, html.EscapeString(tok))
	}

	dn := html.EscapeString(o.provider.DisplayName)
	data := map[string]any{"DisplayName": dn}
	title := locale.User(o.i18n, mattermostUserID, "ycal.caldav.page_title", "Connect {{.DisplayName}}", data)
	intro := locale.User(o.i18n, mattermostUserID, "ycal.caldav.intro_html",
		`Use an <strong>app password</strong> from your <a href="https://yandex.ru/security/app-passwords" target="_blank">account security settings</a> (not your regular login password).`, nil)
	labelEmail := locale.User(o.i18n, mattermostUserID, "ycal.caldav.label_email", "Email", nil)
	labelPassword := locale.User(o.i18n, mattermostUserID, "ycal.caldav.label_app_password", "App password", nil)
	btn := locale.User(o.i18n, mattermostUserID, "ycal.caldav.button_submit", "Connect", nil)

	page := fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>%s</title></head>
<body>
<h1>%s</h1>
<p>%s</p>
<form method="POST" action="">
%s
<label>%s<br><input type="email" name="email" required size="40" autocomplete="username"></label><br><br>
<label>%s<br><input type="password" name="app_password" required size="40" autocomplete="current-password"></label><br><br>
<button type="submit">%s</button>
</form>
</body></html>`,
		title,
		title,
		intro,
		hidden,
		labelEmail,
		labelPassword,
		btn,
	)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(page))
}

func (o *handler) connectPOST(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		msg := locale.Server(o.i18n, "ycal.caldav.err.invalid_form", "invalid form", nil)
		http.Error(w, msg, http.StatusBadRequest)
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
		msg := locale.Server(o.i18n, "ycal.caldav.err.unauthorized_post",
			"Not authorized — open the connect page again from Mattermost (session was not sent with this submit).", nil)
		http.Error(w, msg, http.StatusUnauthorized)
		return
	}
	email := r.Form.Get("email")
	appPassword := r.Form.Get("app_password")
	if email == "" || appPassword == "" {
		msg := locale.User(o.i18n, mattermostUserID, "ycal.caldav.err.missing_credentials",
			"email and app_password are required", nil)
		http.Error(w, msg, http.StatusBadRequest)
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

	dn := html.EscapeString(o.provider.DisplayName)
	successTitle := locale.User(o.i18n, mattermostUserID, "ycal.caldav.success_title", "Connected", nil)
	successBody := locale.User(o.i18n, mattermostUserID, "ycal.caldav.success_body",
		"Connected to {{.DisplayName}}. You can close this window and return to Mattermost.",
		map[string]any{"DisplayName": dn})
	success := fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>%s</title></head>
<body><p>%s</p>
<script>window.close();</script>
</body></html>`, successTitle, successBody)
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
