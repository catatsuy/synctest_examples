package synctest_examples

import (
	"bufio"
	"context"
	"io"
	"testing"
	"testing/synctest"
	"time"
)

type Exec struct {
	r *bufio.Reader
}

func NewExec(pr *io.PipeReader) *Exec {
	return &Exec{r: bufio.NewReader(pr)}
}

func (ex *Exec) Start(ctx context.Context, interval <-chan time.Time, flush func(string), done func(string)) {
	// Reader stops only on EOF.
	go func() {
		for {
			_, _, err := ex.r.ReadLine()
			if err != nil {
				return
			} // EOF only
		}
	}()
	select {
	case <-interval:
		flush("")
	case <-ctx.Done():
		done("")
		// returns, but the reader goroutine is still blocked if writer is not closed
	}
}

func Test_IOPipe_cancel_deadlocks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pr, pw := io.Pipe()
		ex := NewExec(pr)

		ctx, cancel := context.WithCancel(context.Background())
		interval := make(chan time.Time)

		done := make(chan struct{})
		go func() {
			ex.Start(ctx, interval, func(string) {}, func(string) {})
			close(done)
		}()

		// Write one line, then cancel WITHOUT closing pw:
		pw.Write([]byte("abc\n"))
		cancel()
		<-done

		// At test end, synctest panics:
		// panic: deadlock: main bubble goroutine has exited but blocked goroutines remain
		_ = pw
	})
}

func Test_IOPipe_cancel_with_close_ok(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pr, pw := io.Pipe()
		ex := NewExec(pr)

		ctx, cancel := context.WithCancel(context.Background())
		interval := make(chan time.Time)

		done := make(chan struct{})
		go func() {
			ex.Start(ctx, interval, func(string) {}, func(string) {})
			close(done)
		}()

		pw.Write([]byte("abc\n"))
		// Close to send EOF so the reader goroutine exits:
		_ = pw.Close()

		cancel()
		<-done
	})
}
