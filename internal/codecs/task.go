package codecs

import (
	"errors"
	"fmt"
	"io"
	"sync"
)

var errPCMTaskClosed = errors.New("PCM decoder task is closed")

type pcmTaskOperation uint8

const (
	pcmTaskRead pcmTaskOperation = iota
	pcmTaskSeek
)

type pcmTaskRequest struct {
	operation pcmTaskOperation
	buffer    []byte
	offset    int64
	whence    int
}

type pcmTaskResult struct {
	count    int
	position int64
	err      error
}

// pcmTask gives one goroutine exclusive ownership of a decoder. Read and Seek
// remain synchronous to callers, while all decoder work is executed by that
// task and never concurrently.
type pcmTask struct {
	requests  chan pcmTaskRequest
	responses chan pcmTaskResult
	stop      chan struct{}
	done      chan struct{}
	closeOnce sync.Once
	callMu    sync.Mutex
	closeErr  error
}

func newPCMTask(source io.ReadSeeker) *pcmTask {
	task := &pcmTask{
		requests:  make(chan pcmTaskRequest),
		responses: make(chan pcmTaskResult, 1),
		stop:      make(chan struct{}),
		done:      make(chan struct{}),
	}
	go task.run(source)
	return task
}

func (task *pcmTask) Read(destination []byte) (int, error) {
	result, err := task.call(pcmTaskRequest{operation: pcmTaskRead, buffer: destination})
	if err != nil {
		return 0, err
	}
	return result.count, result.err
}

func (task *pcmTask) Seek(offset int64, whence int) (int64, error) {
	result, err := task.call(pcmTaskRequest{operation: pcmTaskSeek, offset: offset, whence: whence})
	if err != nil {
		return 0, err
	}
	return result.position, result.err
}

func (task *pcmTask) Close() error {
	task.closeOnce.Do(func() {
		close(task.stop)
	})
	// Waiting for done is intentional: callers can create the next track only
	// after this decoder goroutine has released its source and exited.
	<-task.done
	return task.closeErr
}

func (task *pcmTask) call(request pcmTaskRequest) (pcmTaskResult, error) {
	task.callMu.Lock()
	defer task.callMu.Unlock()

	select {
	case task.requests <- request:
	case <-task.done:
		return pcmTaskResult{}, errPCMTaskClosed
	}
	select {
	case result := <-task.responses:
		return result, nil
	case <-task.done:
		select {
		case result := <-task.responses:
			return result, nil
		default:
			return pcmTaskResult{}, errPCMTaskClosed
		}
	}
}

func (task *pcmTask) run(source io.ReadSeeker) {
	defer close(task.done)
	if closer, ok := source.(io.Closer); ok {
		defer func() { task.closeErr = closer.Close() }()
	}
	for {
		select {
		case <-task.stop:
			return
		case request := <-task.requests:
			result, panicked := executePCMRequest(source, request)
			task.responses <- result
			if panicked {
				return
			}
		}
	}
}

func executePCMRequest(source io.ReadSeeker, request pcmTaskRequest) (result pcmTaskResult, panicked bool) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result.err = fmt.Errorf("PCM decoder panic: %v", recovered)
			panicked = true
		}
	}()

	switch request.operation {
	case pcmTaskRead:
		result.count, result.err = source.Read(request.buffer)
	case pcmTaskSeek:
		result.position, result.err = source.Seek(request.offset, request.whence)
	default:
		result.err = errors.New("unknown PCM decoder task operation")
	}
	return result, false
}
