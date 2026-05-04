// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Package locale wraps plugin i18n bundles with safe fallbacks when the bundle is nil
// or a translation is missing.
package locale

import (
	"strings"
	"text/template"

	mmi18n "github.com/mattermost/mattermost/server/public/pluginapi/i18n"
)

func fallbackExecute(other string, data map[string]any) string {
	if len(data) == 0 {
		return other
	}
	t, err := template.New("fb").Option("missingkey=zero").Parse(other)
	if err != nil {
		return other
	}
	var b strings.Builder
	if err := t.Execute(&b, data); err != nil {
		return other
	}
	return b.String()
}

// User localizes for the Mattermost user's locale (from profile).
func User(b *mmi18n.Bundle, userID, id, defaultOther string, data map[string]any) string {
	if b == nil {
		return fallbackExecute(defaultOther, data)
	}
	l := b.GetUserLocalizer(userID)
	s := b.LocalizeWithConfig(l, &mmi18n.LocalizeConfig{
		DefaultMessage: &mmi18n.Message{ID: id, Other: defaultOther},
		TemplateData:   data,
	})
	if s == "" {
		return fallbackExecute(defaultOther, data)
	}
	return s
}

// Server localizes using the server's default locale (no user context).
func Server(b *mmi18n.Bundle, id, defaultOther string, data map[string]any) string {
	if b == nil {
		return fallbackExecute(defaultOther, data)
	}
	l := b.GetServerLocalizer()
	s := b.LocalizeWithConfig(l, &mmi18n.LocalizeConfig{
		DefaultMessage: &mmi18n.Message{ID: id, Other: defaultOther},
		TemplateData:   data,
	})
	if s == "" {
		return fallbackExecute(defaultOther, data)
	}
	return s
}
