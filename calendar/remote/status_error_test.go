// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package remote

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/pkg/errors"
)

func TestIsUnauthorized(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "plain 401", err: fmt.Errorf("401 Unauthorized"), want: true},
		{name: "wrapped find principal", err: fmt.Errorf("ycal: FindCurrentUserPrincipal: %w", fmt.Errorf("401 Unauthorized")), want: true},
		{name: "pkg wrap poll chain", err: errors.Wrap(
			errors.Wrap(fmt.Errorf("ycal: FindCurrentUserPrincipal: %w", fmt.Errorf("401 Unauthorized")), "ycal queryRemoteEvents"),
			"ycal PollNotifications",
		), want: true},
		{name: "status error", err: NewStatusError(http.StatusUnauthorized, "auth failed", fmt.Errorf("401 Unauthorized")), want: true},
		{name: "webdav unauthenticated", err: fmt.Errorf("ycal: FindCurrentUserPrincipal: %w", fmt.Errorf("webdav: unauthenticated")), want: true},
		{name: "403", err: fmt.Errorf("403 Forbidden"), want: false},
		{name: "network", err: fmt.Errorf("connection refused"), want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsUnauthorized(tc.err); got != tc.want {
				t.Fatalf("IsUnauthorized(%v)=%v want %v", tc.err, got, tc.want)
			}
		})
	}
}
