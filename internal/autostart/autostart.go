//go:build !windows || cgo

package autostart

import "github.com/emersion/go-autostart"

func Enable(name, displayName string, exec []string) error {
	app := &autostart.App{
		Name:        name,
		DisplayName: displayName,
		Exec:        exec,
	}
	return app.Enable()
}

func Disable(name, displayName string, exec []string) error {
	app := &autostart.App{
		Name:        name,
		DisplayName: displayName,
		Exec:        exec,
	}
	return app.Disable()
}
