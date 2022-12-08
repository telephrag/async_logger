package thread

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"most_logger/alog"
	"runtime"
	"sync"
	"time"
)

func job(r *rand.Rand) int64 {
	val := r.Int63() % 10000
	ng := int64(runtime.NumGoroutine())

	work := md5.Sum([]byte(fmt.Sprint(ng + val)))    // h(m)
	work = md5.Sum([]byte(fmt.Sprint(work, ng+val))) // h(h(m) || m)

	return int64(binary.BigEndian.Uint16(work[:2])) % 10000
}

func RunALog(wg *sync.WaitGroup, eventCount, threadID int) {
	r := rand.New(rand.NewSource(int64(threadID)))

	for i := 0; i < eventCount; i++ {
		alog.Print(fmt.Sprintf("{\"thread\": %d, \"timestamp\": %v, \"result\": %d}\n",
			threadID,
			time.Now(),
			job(r),
		))
	}
	wg.Done()
}

func RunStd(wg *sync.WaitGroup, eventCount, threadID int) {
	r := rand.New(rand.NewSource(int64(threadID * 2)))

	for i := 0; i < eventCount; i++ {
		log.Printf("{\"thread\": %d, \"timestamp\": %v, \"result\": %d}\n",
			threadID,
			time.Now(),
			job(r),
		)
	}
	wg.Done()
}
