package ycal

import (
	"path"
	"strings"
)

func joinCalendarObjectPath(calPath, baseName string) string {
	calPath = strings.TrimSuffix(calPath, "/")
	baseName = strings.TrimPrefix(path.Base(baseName), "/")
	return calPath + "/" + baseName
}

func isAbsoluteCalPath(p string) bool {
	return strings.HasPrefix(p, "/")
}

func safeICSFileName(uid string) string {
	uid = strings.ReplaceAll(uid, "/", "_")
	uid = strings.ReplaceAll(uid, "\\", "_")
	if !strings.HasSuffix(strings.ToLower(uid), ".ics") {
		return uid + ".ics"
	}
	return uid
}
