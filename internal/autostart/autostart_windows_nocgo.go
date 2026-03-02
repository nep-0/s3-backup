//go:build windows && !cgo
// +build windows,!cgo

package autostart

import "errors"

var ErrUnsupported = errors.New("autostart unsupported on windows without cgo")

func Enable(name, displayName string, exec []string) error {
	return ErrUnsupported
}

func Disable(name, displayName string, exec []string) error {
	return ErrUnsupported
}
