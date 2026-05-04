// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package ycal

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/config"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/bot"
)

const Kind = "ycal"

type impl struct {
	conf   *config.Config
	logger bot.Logger
}

func init() {
	remote.Makers[Kind] = NewRemote
}

func NewRemote(conf *config.Config, logger bot.Logger) remote.Remote {
	return &impl{
		conf:   conf,
		logger: logger,
	}
}

// EncodeCalDAVCredentials implements remote.CaldavPasswordEncoder.
func (r *impl) EncodeCalDAVCredentials(email, appPassword string) string {
	return EncodeCalDAVCredentials(email, appPassword)
}

func (r *impl) MakeUserClient(ctx context.Context, oauthToken *oauth2.Token, mattermostUserID string, poster bot.Poster, userTokenHelpers remote.UserTokenHelpers) remote.Client {
	oconf := r.NewOAuth2Config()
	token, err := userTokenHelpers.RefreshAndStoreToken(oauthToken, oconf, mattermostUserID)
	if err != nil {
		r.logger.Warnf("ycal: token refresh for user %s: %s", mattermostUserID, err.Error())
		return &client{}
	}

	email, pass, err := decodeCalDAVCredentials(token.AccessToken)
	if err != nil {
		r.logger.Warnf("ycal: bad stored credentials for %s: %v", mattermostUserID, err)
		return &client{}
	}

	rt := newBasicAuthRoundTripper(email, pass, http.DefaultTransport)
	hc := &http.Client{Transport: rt}

	c := &client{
		conf:           r.conf,
		ctx:            ctx,
		httpClient:     hc,
		Logger:         r.logger,
		mmUserID:       mattermostUserID,
		tokenHelpers:   userTokenHelpers,
		oauthToken:     token,
		oauthConfig:    oconf,
		email:          email,
		appPassword:    pass,
		yandexEndpoint: yandexCalDAVRoot,
	}
	return c
}

func (r *impl) MakeSuperuserClient(_ context.Context) (remote.Client, error) {
	return nil, remote.ErrSuperUserClientNotSupported
}

func (r *impl) NewOAuth2Config() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     "caldav-not-oauth",
		ClientSecret: "not-used",
		RedirectURL:  r.conf.PluginURL + config.FullPathOAuth2Redirect,
		Scopes:       []string{},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://invalid.invalid/oauth",
			TokenURL: "https://invalid.invalid/token",
		},
	}
}

func (r *impl) CheckConfiguration(cfg config.StoredConfig) error {
	if cfg.EncryptionKey == "" {
		return fmt.Errorf("encryption key cannot be empty")
	}
	return nil
}

func (r *impl) HandleWebhook(w http.ResponseWriter, req *http.Request) []*remote.Notification {
	w.WriteHeader(http.StatusOK)
	return nil
}
