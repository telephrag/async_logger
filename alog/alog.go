package alog

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"
)

type ALogger struct {
	records  chan []byte
	buff     bufio.Writer
	queueLen int32         // amount of items currently awaiting to be written to `records`
	done     chan struct{} // used for signaling inside `alog.Run()` as well as `inactive`
	inactive chan struct{}
}

var std *ALogger = New(os.Stdout, 16, 1024)

// Reinitializes `alog` with given set of parameters.
// Parameters:
// output 		-- output location that satisfies `io.Writer`
// recordsCount -- amount of records stored in channel simultaneosuly
//		ready for writing to output
// bufferSize   -- size of underlying buffer used for writing to output
func Init(output io.Writer, recordsCount, bufferSize int) {
	std = New(output, recordsCount, bufferSize)
}

func New(output io.Writer, recordsCount, bufferSize int) *ALogger {
	al := &ALogger{
		records:  make(chan []byte),
		buff:     *bufio.NewWriterSize(output, bufferSize),
		done:     make(chan struct{}, 1),
		inactive: make(chan struct{}, 1),
	}
	al.inactive <- struct{}{}
	return al
}

func (al *ALogger) write(s []byte) (int, error) {
	bytesWritten := 0
	for bytesWritten < len(s) {
		bw, err := al.buff.Write(s[bytesWritten:])
		if err != nil {
			return bytesWritten, err
		}
		bytesWritten += bw
	}
	return bytesWritten, nil
}

// Runs the logger. Must be executed in a separate thread.
// Requires calling `Flush()` or `Finish()` after exiting. Later is preferred.
func (al *ALogger) Run() error {
	<-al.inactive

	var stow []byte
	for {
		select {
		case <-al.done: // occurs only on call to `alog.Panic()` or `alog.Fatal()`
			al.inactive <- struct{}{}
			return nil
		case r := <-al.records:
			r = append(stow, r...) // from rough testing calling `bufio.Flush()` would be more expensive
			if bw, err := al.write(r); err != nil {
				if err == io.ErrShortWrite {
					stow = r[bw:]
				} else {
					// Errors that are not `io.ErrShortWrite` should be handled outside
					// of this method. Than `alog` can be restarted by calling `Run()` again.
					return err
				}
			} else {
				stow = []byte("")
			}
		}
	}
}

// Flushes underlying buffer.
func (al *ALogger) Flush() {
	al.buff.Flush()
}

// Checks if `alog` is running and accepting records for logging
// inside `Run()`.
func (al *ALogger) IsActive() bool {
	return len(al.inactive) == 0
}

// Waits until there is nothing left to log.
// Meant to be deffered to gracefully shut down the application.
// It's your responsibility to make sure that no more writes to `al.records` will occur.
// Calling while `Run()` hasn't finished might result in loss of data.
func (al *ALogger) Finish() {
	for atomic.LoadInt32(&al.queueLen) != 0 || len(al.records) != 0 {
	}
	// To avoid last few records not logging. Seems to not occur with higher events and threads count.
	// Idk, how to do it without such bandaids.
	time.Sleep(time.Millisecond * 100)
	al.buff.Flush()
}

// Prints log message to `io.Writer` set via `Init()` or to `os.Stdout`
func (al *ALogger) Print(s ...any) {
	r := []byte(fmt.Sprint(s...))
	atomic.AddInt32(&al.queueLen, 1)
	al.records <- r
	atomic.AddInt32(&al.queueLen, -1)
}

func (al *ALogger) Println(s ...any) {
	r := []byte(fmt.Sprint(s...) + "\n")
	atomic.AddInt32(&al.queueLen, 1)
	al.records <- r
	atomic.AddInt32(&al.queueLen, -1)
}

func (al *ALogger) Printf(format string, s ...any) {
	r := []byte(fmt.Sprintf(format, s...) + "\n")
	atomic.AddInt32(&al.queueLen, 1)
	al.records <- r
	atomic.AddInt32(&al.queueLen, -1)
}

// If `Run()` is writing makes it finish writing current record.
// Then writes given message and calls `os.Exit(1)`.
func (al *ALogger) Fatal(s ...any) {
	r := fmt.Sprint(s...)
	al.done <- struct{}{} // to make sure that write isn't happening inside `Run()` at the moment
	<-al.inactive
	al.write([]byte(fmt.Sprint("fatal: ", r)))
	al.buff.Flush()
	os.Exit(1)
}

func (al *ALogger) Fatalln(s ...any) {
	r := fmt.Sprint(s...)
	al.done <- struct{}{} // to make sure that write isn't happening inside `Run()` at the moment
	<-al.inactive
	al.write([]byte(fmt.Sprint("fatal: ", r, "\n")))
	al.buff.Flush()
	os.Exit(1)
}

func (al *ALogger) Fatalf(format string, s ...any) {
	r := fmt.Sprint(s...)
	al.done <- struct{}{} // to make sure that write isn't happening inside `Run()` at the moment
	<-al.inactive
	al.write([]byte(fmt.Sprintf(format, r)))
	al.buff.Flush()
	os.Exit(1)
}

// Same as `Fatal()` but calls `panic()` instead of `os.Exit(1)`
func (al *ALogger) Panic(s ...any) {
	r := fmt.Sprint(s...)
	al.done <- struct{}{} // to make sure that write isn't happening inside `Run()` at the moment
	<-al.inactive
	al.write([]byte(fmt.Sprint("panic: ", r)))
	al.buff.Flush()
	panic(s)
}

func (al *ALogger) Panicln(s ...any) {
	r := fmt.Sprint(s...)
	al.done <- struct{}{} // to make sure that write isn't happening inside `Run()` at the moment
	<-al.inactive
	al.write([]byte(fmt.Sprint("panic: ", r, "\n")))
	al.buff.Flush()
	panic(s)
}

func (al *ALogger) Panicf(format string, s ...any) {
	r := fmt.Sprint(s...)
	al.done <- struct{}{} // to make sure that write isn't happening inside `Run()` at the moment
	<-al.inactive
	al.write([]byte(fmt.Sprintf(format, r)))
	al.buff.Flush()
	panic(s)
}

// Runs the logger. Should be executed in a separate thread.
func Run() error {
	return std.Run()
}

// Flushes underlying buffer.
func Flush() {
	std.Flush()
}

// Checks if `alog` is running and accepting records for logging
// inside `Run()`.
func IsActive() bool {
	return std.IsActive()
}

// Waits until there is nothing left to log.
// Meant to be deffered to gracefully shut down the application.
// Calling while `Run()` hasn't finished might result in loss of data.
func Finnish() {
	std.Finish()
}

// Prints log message to `io.Writer` set via `Init()` or to `os.Stdout`
func Print(s ...any) {
	go std.Print(s...)
}

func Println(s ...any) {
	go std.Println(s...)
}

func Printf(format string, s ...any) {
	go std.Printf(format, s...)
}

// If `Run()` is writing makes it finish writing current record.
// Then writes given message and calls `os.Exit(1)`.
func Fatal(s ...any) {
	std.Fatal(s...)
}

func Fatalln(s ...any) {
	std.Fatalln(s...)
}

func Fatalf(format string, s ...any) {
	std.Fatalf(format, s...)
}

// Same as `Fatal()` but calls `panic()` instead of `os.Exit(1)`
func Panic(s ...any) {
	std.Panic(s...)
}

func Panicln(s ...any) {
	std.Panicln(s...)
}

func Panicf(format string, s ...any) {
	std.Panicf(format, s...)
}
