// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package caldavconnect

import "testing"

func TestValidConnectTokenHex(t *testing.T) {
	t.Parallel()
	good := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if !validConnectTokenHex(good) {
		t.Fatal("expected valid token")
	}
	cases := []string{
		"",
		"short",
		good + "0",
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdeg", // bad char
	}
	for _, s := range cases {
		if validConnectTokenHex(s) {
			t.Fatalf("expected invalid: %q", s)
		}
	}
}
