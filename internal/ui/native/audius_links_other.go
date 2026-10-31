//go:build !windows

package native

import (
	"os/exec"
	"runtime"
)

func openExternalURL(raw string) error {
	if err := validateAudiusLink(raw); err != nil {
		return err
	}
	command := "xdg-open"
	if runtime.GOOS == "darwin" {
		command = "open"
	}
	// #nosec G204 -- fixed OS launcher, validated HTTPS Audius URL passed as one argument, no shell.
	cmd := exec.Command(command, raw)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
