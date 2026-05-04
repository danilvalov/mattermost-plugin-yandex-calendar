package ycal

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type storedCreds struct {
	Email string `json:"e"`
	Pass  string `json:"p"`
}

// EncodeCalDAVCredentials stores email + app password in the oauth2.Token.AccessToken field.
func EncodeCalDAVCredentials(email, appPassword string) string {
	b, _ := json.Marshal(storedCreds{Email: email, Pass: appPassword})
	return base64.StdEncoding.EncodeToString(b)
}

func decodeCalDAVCredentials(accessToken string) (email, password string, err error) {
	raw, err := base64.StdEncoding.DecodeString(accessToken)
	if err != nil {
		return "", "", err
	}
	var sc storedCreds
	if err := json.Unmarshal(raw, &sc); err != nil {
		return "", "", err
	}
	if sc.Email == "" || sc.Pass == "" {
		return "", "", fmt.Errorf("missing credentials")
	}
	return sc.Email, sc.Pass, nil
}
