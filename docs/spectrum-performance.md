# Spectrum performance check (2026-10-27)

Measured on Windows amd64, AMD Ryzen 5 4600G, using Go benchmarks.

* FFT: `BenchmarkAnalyzerRead` processes one second of 48 kHz stereo PCM in
  5.24–5.59 ms, with zero allocations per iteration. This measures analysis
  throughput, not output latency or audiovisual synchronization.
* Drawing preparation: `BenchmarkSpectrumBars` covers 32 full-height bars
  (180 pixels), reusing Gio operations. Before: 31.5–31.7 microseconds per
  iteration. After: 23.0–23.2 microseconds, about 27% less CPU time.
  Both versions report zero allocations per iteration. These measurements
  exclude GPU rendering and the rest of the window layout.

Separator color is now submitted once per bar instead of once per separator.
The appearance and rectangular drawing primitives are preserved. Combining
separators into a general path was measured and discarded because it increased
CPU preparation time.

Bar and VU decay now stop at the measured target, avoiding alternating dips
under a steady input. Peak hold uses elapsed time (260 ms), rather than a
frame count. Delayed frames use elapsed time capped at 200 ms. Tests cover
peak timing at 30, 60 and 120 FPS and decay after a delayed frame.

Validation: `go test ./...`, `go vet ./...`, and a normal Go executable build
passed. Race testing could not start because CGO is disabled in this environment.
End-to-end frame pacing, GPU cost, and audio/display synchronization still need
an interactive measurement; no measured FPS improvement is claimed here.

Reproduce the measurements:

```powershell
go test ./internal/dsp -run '^$' -bench BenchmarkAnalyzerRead -benchmem -count=3
go test ./internal/ui/native -run '^$' -bench BenchmarkSpectrumBars -benchmem -count=3
```
