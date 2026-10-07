package native

import (
	"strings"
	"testing"
)

func TestLoadSkin(t *testing.T) {
	skin, err := loadSkin(strings.NewReader(`{"version":1,"name":"TEST","colors":{"accent":"#123456","channelRight":"#abcdef80"},"metrics":{"spacing":12}}`))
	if err != nil {
		t.Fatal(err)
	}
	palette := paletteFor(skin)
	if palette.Accent.R != 0x12 || palette.Accent.G != 0x34 || palette.Accent.B != 0x56 {
		t.Fatalf("unexpected accent: %#v", palette.Accent)
	}
	if palette.ChannelRight.A != 0x80 {
		t.Fatalf("unexpected channel alpha: %#v", palette.ChannelRight)
	}
	if palette.Background.A != 0xff {
		t.Fatal("missing colors were not inherited from the base skin")
	}
}

func TestLoadSkinRejectsUnknownColor(t *testing.T) {
	_, err := loadSkin(strings.NewReader(`{"version":1,"name":"TEST","colors":{"surprise":"#123456"}}`))
	if err == nil {
		t.Fatal("expected unknown color to be rejected")
	}
}
