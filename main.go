package main

import (
	"flag"
	"fmt"
	"log"
	"most_logger/alog"
	"most_logger/thread"
	"os"
	"sync"
	"time"
)

var eventCount, threadCount, recordCount, bufferSize int

func init() {
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))

	flag.IntVar(&eventCount, "e", 50, "Amount of events thread emits. Default 50.")
	flag.IntVar(&threadCount, "t", 7, "Amount of threads. Default 7.")
	flag.IntVar(&recordCount, "r", 32, "Amount of records. Default 32.")
	flag.IntVar(&bufferSize, "b", 4096, "Buffer size for async logger. Default 4096.")

	flag.Parse()

	// bufferSize = 100

	f := getLogFile("most.log")
	alog.Init(f, recordCount, bufferSize)
	log.SetOutput(f)
}

func getLogFile(path string) *os.File {
	_, err := os.Create(path) // if not called for some reasone OpenFile will err
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.OpenFile(path, os.O_WRONLY, os.ModeAppend)
	if err != nil {
		_, err := os.Create(path)
		if err != nil {
			log.Fatalln(err)
		}
	}
	return f
}

func main() {

	var wg sync.WaitGroup

	// standard library
	start := time.Now()
	wg.Add(threadCount)
	for i := 0; i < threadCount; i++ {
		go thread.RunStd(&wg, eventCount, i)
	}
	wg.Wait()
	fmt.Println("log:  ", time.Since(start))
	log.Println()

	// alog
	startAsync := time.Now()
	go func() {
		if err := alog.Run(); err != nil {
			log.Fatal(err)
		}
	}()
	defer alog.Finnish()
	wg.Add(threadCount)
	for i := 0; i < threadCount; i++ {
		go thread.RunALog(&wg, eventCount, i)
	}
	wg.Wait()
	fmt.Println("alog: ", time.Since(startAsync))
}
