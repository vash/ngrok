//go:build !release && !autoupdate
// +build !release,!autoupdate

package client

import (
	"ngrok/pkg/client/mvc"
)

// no auto-updating in debug mode
func autoUpdate(state mvc.State, token string) {
}
