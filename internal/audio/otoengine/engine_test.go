package otoengine

import (
	"testing"
	"time"
)

func TestByteOffset(t *testing.T) {
	offset, err := byteOffset(1500*time.Millisecond, 44_100, 4)
	if err != nil {
		t.Fatal(err)
	}
	if want := int64(264_600); offset != want {
		t.Fatalf("byteOffset() = %d, want %d", offset, want)
	}
}

func TestByteOffsetRejectsNegativePosition(t *testing.T) {
	if _, err := byteOffset(-time.Second, 44_100, 4); err == nil {
		t.Fatal("byteOffset() accepted a negative position")
	}
}
