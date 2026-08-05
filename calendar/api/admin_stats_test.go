// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/store"
)

func TestAdminStats(t *testing.T) {
	api, mockStore, _, _, mockPluginAPI, _, _, _ := GetMockSetup(t)

	tests := []struct {
		name       string
		setup      func(req *http.Request)
		assertions func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name:  "Missing Mattermost-User-Id",
			setup: func(req *http.Request) {},
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, rec.Result().StatusCode)
			},
		},
		{
			name: "Not authorized admin",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				api.AdminUserIDs = ""
				mockPluginAPI.EXPECT().IsSysAdmin(MockUserID).Return(false, nil).Times(1)
			},
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusForbidden, rec.Result().StatusCode)
			},
		},
		{
			name: "IsSysAdmin error",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				api.AdminUserIDs = ""
				mockPluginAPI.EXPECT().IsSysAdmin(MockUserID).Return(false, errors.New("sysadmin check failed")).Times(1)
			},
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, rec.Result().StatusCode)
			},
		},
		{
			name: "GetStats error",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				api.AdminUserIDs = ""
				mockPluginAPI.EXPECT().IsSysAdmin(MockUserID).Return(true, nil).Times(1)
				mockStore.EXPECT().GetStats().Return(nil, errors.New("store error")).Times(1)
			},
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, rec.Result().StatusCode)
			},
		},
		{
			name: "Success as sysadmin",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				api.AdminUserIDs = ""
				mockPluginAPI.EXPECT().IsSysAdmin(MockUserID).Return(true, nil).Times(1)
				mockStore.EXPECT().GetStats().Return(&store.Stats{
					ConnectedUsers: 3,
					InactiveUsers:  1,
					Subscriptions:  2,
				}, nil).Times(1)
			},
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
				var stats store.Stats
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&stats))
				assert.Equal(t, uint64(3), stats.ConnectedUsers)
				assert.Equal(t, uint64(1), stats.InactiveUsers)
				assert.Equal(t, uint64(2), stats.Subscriptions)
			},
		},
		{
			name: "Success via AdminUserIDs",
			setup: func(req *http.Request) {
				req.Header.Set(MMUserIDHeader, MockUserID)
				api.AdminUserIDs = MockUserID
				mockStore.EXPECT().GetStats().Return(&store.Stats{ConnectedUsers: 1}, nil).Times(1)
			},
			assertions: func(t *testing.T, rec *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, rec.Result().StatusCode)
				var stats store.Stats
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&stats))
				assert.Equal(t, uint64(1), stats.ConnectedUsers)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/stats", nil)
			tt.setup(req)
			rec := httptest.NewRecorder()
			api.adminStats(rec, req)
			tt.assertions(t, rec)
		})
	}
}
