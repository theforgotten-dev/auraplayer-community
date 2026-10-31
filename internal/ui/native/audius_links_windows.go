//go:build windows

package native

import "golang.org/x/sys/windows"

func openExternalURL(raw string) error {
	if err := validateAudiusLink(raw); err != nil {
		return err
	}
	value, err := windows.UTF16PtrFromString(raw)
	if err != nil {
		return err
	}
	return windows.ShellExecute(0, nil, value, nil, nil, 1)
}
