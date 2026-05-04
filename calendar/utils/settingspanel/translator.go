// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package settingspanel

import (
	"strings"
	"text/template"
)

// Translator localizes UI copy for a Mattermost user. Nil means template-expand defaults only.
type Translator func(userID, messageID, defaultOther string, templateData map[string]any) string

func expandDefault(other string, data map[string]any) string {
	if len(data) == 0 {
		return other
	}
	t, err := template.New("sp").Option("missingkey=zero").Parse(other)
	if err != nil {
		return other
	}
	var b strings.Builder
	if err := t.Execute(&b, data); err != nil {
		return other
	}
	return b.String()
}

// T returns the localized string or expands defaultOther when tr is nil or localization is empty.
func (tr Translator) T(userID, messageID, defaultOther string, templateData map[string]any) string {
	if tr == nil {
		return expandDefault(defaultOther, templateData)
	}
	s := tr(userID, messageID, defaultOther, templateData)
	if strings.TrimSpace(s) == "" {
		return expandDefault(defaultOther, templateData)
	}
	return s
}
