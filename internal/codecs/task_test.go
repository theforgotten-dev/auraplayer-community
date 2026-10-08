package codecs

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestPCMTaskReadsAndSeeks(t *testing.T) {
	task := newPCMTask(bytes.NewReader([]byte("abcdef")))
	defer task.Close()

	buffer := make([]byte, 3)
	if read, err := task.Read(buffer); err != nil || read != 3 || string(buffer) != "abc" {
		t.Fatalf("Read() = (%d, %q, %v), want (3, abc, nil)", read, buffer, err)
	}
	if position, err := task.Seek(1, io.SeekStart); err != nil || position != 1 {
		t.Fatalf("Seek() = (%d, %v), want (1, nil)", position, err)
	}
	if read, err := task.Read(buffer); err != nil || read != 3 || string(buffer) != "bcd" {
		t.Fatalf("Read() after seek = (%d, %q, %v), want (3, bcd, nil)", read, buffer, err)
	}
}

func TestPCMTaskRejectsOperationsAfterClose(t *testing.T) {
	task := newPCMTask(bytes.NewReader(nil))
	if err := task.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := task.Read(make([]byte, 1)); !errors.Is(err, errPCMTaskClosed) {
		t.Fatalf("Read() error = %v, want %v", err, errPCMTaskClosed)
	}
}

func TestPCMTaskContainsDecoderPanic(t *testing.T) {
	task := newPCMTask(panicReadSeeker{})
	defer task.Close()
	if _, err := task.Read(make([]byte, 1)); err == nil || !strings.Contains(err.Error(), "decoder panic") {
		t.Fatalf("Read() error = %v, want recovered decoder panic", err)
	}
}

func TestPCMTaskStopsBeforeNextTrackTaskStarts(t *testing.T) {
	var previousDone <-chan struct{}
	for track := 0; track < 32; track++ {
		if previousDone != nil {
			select {
			case <-previousDone:
			default:
				t.Fatalf("decoder task for track %d is still running", track-1)
			}
		}

		task := newPCMTask(bytes.NewReader([]byte("track")))
		buffer := make([]byte, 5)
		if _, err := task.Read(buffer); err != nil {
			t.Fatalf("track %d: %v", track, err)
		}
		previousDone = task.done
		if err := task.Close(); err != nil {
			t.Fatalf("close track %d: %v", track, err)
		}
	}

	select {
	case <-previousDone:
	default:
		t.Fatal("last decoder task is still running")
	}
}

func TestPCMTaskClosesOwnedDecoderBeforeExit(t *testing.T) {
	source := &closingReadSeeker{Reader: bytes.NewReader([]byte("track"))}
	task := newPCMTask(source)
	if err := task.Close(); err != nil {
		t.Fatal(err)
	}
	if !source.closed {
		t.Fatal("decoder source remained open after task exit")
	}
}

type closingReadSeeker struct {
	*bytes.Reader
	closed bool
}

func (source *closingReadSeeker) Close() error {
	source.closed = true
	return nil
}

type panicReadSeeker struct{}

func (panicReadSeeker) Read([]byte) (int, error) { panic("malformed stream") }
func (panicReadSeeker) Seek(int64, int) (int64, error) {
	return 0, nil
}
