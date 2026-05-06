// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package views

import (
	"testing"
	"time"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
	"github.com/stretchr/testify/require"
)

func TestMarkdownToHTMLEntities(t *testing.T) {
	for _, testCase := range []struct {
		description    string
		inputstring    string
		expectedOutput string
	}{
		{
			description:    "with asterisk",
			inputstring:    "**bold text**",
			expectedOutput: "&#42;&#42;bold text&#42;&#42;",
		},
		{
			description:    "normal string",
			inputstring:    "normal string",
			expectedOutput: "normal string",
		},
		{
			description:    "with braces",
			inputstring:    "[square](round)",
			expectedOutput: "&#91;square&#93;&#40;round&#41;",
		},
		{
			description:    "with underscore",
			inputstring:    "text_test",
			expectedOutput: "text&#95;test",
		},
		{
			description:    "withbacktick",
			inputstring:    "`test`",
			expectedOutput: "&#96;test&#96;",
		},
		{
			description:    "with greater and less than",
			inputstring:    "<test>",
			expectedOutput: "&#60;test&#62;",
		},
		{
			description:    "with backslash",
			inputstring:    "test \\ text",
			expectedOutput: "test &#92; text",
		},
		{
			description:    "URL 1",
			inputstring:    "www.example.com",
			expectedOutput: "www&#46;example&#46;com",
		},
		{
			description:    "URL 2",
			inputstring:    "https://example.com",
			expectedOutput: "https&#58;&#47;&#47;example&#46;com",
		},
		{
			description:    "strike through",
			inputstring:    "~~strike~~",
			expectedOutput: "&#126;&#126;strike&#126;&#126;",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			res := MarkdownToHTMLEntities(testCase.inputstring)
			require.EqualValues(t, testCase.expectedOutput, res)
		})
	}
}

func TestLinkifyAndEscapeText(t *testing.T) {
	input := "Join\nhttps://telemost.360.yandex.ru/j/8510081139\nContact: team@example.com."
	got := LinkifyAndEscapeText(input)

	require.Contains(t, got, "[https://telemost.360.yandex.ru/j/8510081139](https://telemost.360.yandex.ru/j/8510081139)")
	require.Contains(t, got, "[team@example.com](mailto:team@example.com)")
	require.Contains(t, got, ".")
}

func TestRenderUpcomingEventAsAttachmentWithTimeFormat_AddsDescriptionField(t *testing.T) {
	event := &remote.Event{
		Subject: "Standup",
		Body: &remote.ItemBody{
			Content: "Присоединиться Yandex Telemost\nhttps://telemost.360.yandex.ru/j/8510081139",
		},
		Start: remote.NewDateTime(time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC), "UTC"),
		End:   remote.NewDateTime(time.Date(2026, 5, 6, 12, 30, 0, 0, time.UTC), "UTC"),
	}

	_, attachment, err := RenderUpcomingEventAsAttachmentWithTimeFormat(event, "UTC", false, nil, "u1")
	require.NoError(t, err)
	require.NotNil(t, attachment)

	require.Len(t, attachment.Fields, 1)
	require.Equal(t, "Description", attachment.Fields[0].Title)
	require.Contains(t, attachment.Fields[0].Value, "Присоединиться Yandex Telemost")
	require.Contains(t, attachment.Fields[0].Value, "[https://telemost.360.yandex.ru/j/8510081139](https://telemost.360.yandex.ru/j/8510081139)")
}
