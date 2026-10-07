//go:build windows

package main

// Regenerate the Windows application resources after changing the source icon.
//go:generate go run github.com/tc-hib/go-winres@v0.3.3 simply --arch amd64,arm64 --out rsrc --manifest gui --icon assets/aura-icon.png --product-name "Aura Music Player" --file-description "Aura Music Player" --original-filename "aura-native.exe"
