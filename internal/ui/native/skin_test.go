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

func TestLoadSkinRejectsTrailingJSON(t *testing.T) {
	_, err := loadSkin(strings.NewReader(`{"version":1,"name":"TEST"} {"version":1,"name":"SECOND"}`))
	if err == nil {
		t.Fatal("expected a second JSON value to be rejected")
	}
}

func TestLoadSkinEnforcesSizeLimit(t *testing.T) {
	prefix := `{"version":1,"name":"TEST"}`
	oversized := prefix + strings.Repeat(" ", maxSkinSize-len(prefix)+1)
	_, err := loadSkin(strings.NewReader(oversized))
	if err == nil {
		t.Fatal("expected an oversized skin to be rejected")
	}
}

func TestLoadSkinRejectsControlCharactersInMetadata(t *testing.T) {
	_, err := loadSkin(strings.NewReader("{\"version\":1,\"name\":\"TEST\\nINJECTED\"}"))
	if err == nil {
		t.Fatal("expected control characters in the skin name to be rejected")
	}
}
