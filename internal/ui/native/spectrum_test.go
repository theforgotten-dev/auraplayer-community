package native

import (
	"image"
	"image/color"
	"math"
	"testing"
	"time"

	"gioui.org/op"
)

func TestSpectrumDecayIndependentOfFrameRate(t *testing.T) {
	var results []float64
	for _, fps := range []int{30, 60, 120} {
		now := time.Unix(1, 0)
		ui := UI{lastAnimTime: now}
		ui.peaksLeft[0], ui.peaksRight[0] = 1, 1
		ui.peakHoldLeft[0], ui.peakHoldRight[0] = 260*time.Millisecond, 260*time.Millisecond
		for frame := 1; frame <= fps; frame++ {
			ui.animateSpectrum(now.Add(time.Duration(frame)*time.Second/time.Duration(fps)), false)
		}
		results = append(results, ui.peaksLeft[0])
		if math.Abs(ui.peaksLeft[0]-ui.peaksRight[0]) > 1e-9 {
			t.Fatal("channel peak timing differs")
		}
	}
	for _, peak := range results {
		if math.Abs(peak-0.408) > 0.03 {
			t.Fatalf("peak after one second = %v, want about 0.408 across frame rates; results %v", peak, results)
		}
	}
}

func TestSpectrumDecayAfterDelayedFrame(t *testing.T) {
	now := time.Unix(1, 0)
	ui := UI{lastAnimTime: now, levelLeft: 1, levelRight: 1}
	ui.visualLeft[0], ui.visualRight[0] = 1, 1
	ui.animateSpectrum(now.Add(time.Second), false)
	if math.Abs(ui.visualLeft[0]-0.64) > 1e-9 || math.Abs(ui.levelLeft-0.56) > 1e-9 {
		t.Fatalf("delayed frame did not use bounded elapsed time: bar %v, level %v", ui.visualLeft[0], ui.levelLeft)
	}
}

func BenchmarkSpectrumBars(b *testing.B) {
	var ops op.Ops
	foreground := color.NRGBA{R: 100, G: 220, A: 255}
	background := color.NRGBA{A: 255}
	b.ReportAllocs()
	for b.Loop() {
		ops.Reset()
		for i := 0; i < 32; i++ {
			fillSegmentedBar(&ops, image.Rect(i*12, 0, i*12+10, 180), foreground, background, 180)
		}
	}
}
