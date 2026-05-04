// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package ycal

import (
	"context"
	"net/http"
	"net/url"
	"sync"

	"golang.org/x/oauth2"

	"github.com/emersion/go-webdav/caldav"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/config"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/bot"
)

const yandexCalDAVRoot = "https://caldav.yandex.ru"

type client struct {
	conf           *config.Config
	ctx            context.Context
	httpClient     *http.Client
	Logger         bot.Logger
	mmUserID       string
	tokenHelpers   remote.UserTokenHelpers
	oauthToken     *oauth2.Token
	oauthConfig    *oauth2.Config
	email          string
	appPassword    string
	yandexEndpoint string

	calMu         sync.Mutex
	cachedCalDAV  *caldav.Client
	cachedCalPath string
}

func newBasicAuthRoundTripper(user, pass string, wrapped http.RoundTripper) http.RoundTripper {
	return &basicAuthRoundTripper{user: user, pass: pass, wrapped: wrapped}
}

type basicAuthRoundTripper struct {
	user, pass string
	wrapped    http.RoundTripper
}

func (b *basicAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(b.user, b.pass)
	return b.wrapped.RoundTrip(req)
}

func (c *client) CallJSON(method, url string, in, out any) (responseData []byte, err error) {
	return nil, remote.ErrNotImplemented
}

func (c *client) CallFormPost(method, url string, in url.Values, out any) (responseData []byte, err error) {
	return nil, remote.ErrNotImplemented
}
