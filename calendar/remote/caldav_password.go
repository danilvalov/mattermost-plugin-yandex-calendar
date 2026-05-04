// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package remote

// CaldavPasswordEncoder is implemented by calendar remotes that use app-password auth.
type CaldavPasswordEncoder interface {
	EncodeCalDAVCredentials(email, appPassword string) string
}
