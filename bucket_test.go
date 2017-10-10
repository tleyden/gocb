package gocb

import (
	"fmt"
	"log"
	"regexp"
	"runtime"
	"testing"
	"time"

	"flag"
	"os"

	"gopkg.in/couchbaselabs/gojcbmock.v1"
)

var globalBucket *Bucket
var globalMock *gojcbmock.Mock

func TestMain(m *testing.M) {
	flag.Parse()
	mpath, err := gojcbmock.GetMockPath()
	if err != nil {
		panic(err.Error())
	}

	globalMock, err = gojcbmock.NewMock(mpath, 4, 1, 64, []gojcbmock.BucketSpec{
		{Name: "default", Type: gojcbmock.BCouchbase},
		{Name: "memd", Type: gojcbmock.BMemcached},
	}...)

	if err != nil {
		panic(err.Error())
	}

	connStr := fmt.Sprintf("http://127.0.0.1:%d", globalMock.EntryPort)

	cluster, err := Connect(connStr)

	if err != nil {
		panic(err.Error())
	}

	globalBucket, err = cluster.OpenBucket("default", "")

	if err != nil {
		panic(err.Error())
	}

	os.Exit(m.Run())
}

// Repro attempt for https://issues.couchbase.com/browse/GOCBC-236
func TestReproduceGOCBC236(t *testing.T) {

	SetLogger(VerboseStdioLogger())

	numIterations := 1
	for i := 0; i < numIterations; i++ {

		cluster, err := Connect("http://localhost:8091")
		if err != nil {
			t.Fatalf("Failed to connect to couchbase: %v", err)
		}

		goCBBucket, err := cluster.OpenBucket("test_data_bucket", "password")
		if err != nil {
			t.Fatalf("Failed to open bucket: %v", err)
		}

		// Set view timeout, in case this is relevant (probably not)
		goCBBucket.SetViewTimeout(time.Second * 100)

		doc := map[string]interface{}{}
		doc["foo"] = "bar"
		key := fmt.Sprintf("TestReproduceGOCBC236-%d", i)
		goCBBucket.Upsert(key, doc, 1)

		if err := goCBBucket.Close(); err != nil {
			t.Fatalf("Failed to close bucket: %v", err)
		}

	}

	passed := false
	maxRetries := 30 // 30 seconds
	sleepDurationPerRetry := time.Second
	var buf []byte

	for i := 0; i < maxRetries; i++ {

		buf = make([]byte, 1<<20)

		runtime.Stack(buf, true)

		// Make sure gocb is not in any of the stacks
		r := regexp.MustCompile("gocb")
		if !r.Match(buf) {

			log.Printf("Noo gocb in goroutine stacks.  Setting passed = true")

			passed = true
			break
		}

		log.Printf("Found gocb in goroutine stacks, going to pause and retry")

		// Sleep for a while and retry
		time.Sleep(sleepDurationPerRetry)

	}

	if !passed {

		log.Printf("--------------------------- Dump Stacks ----------------------------------")
		log.Printf("%s", buf)

		t.Fatalf("Found gocb in the goroutine stack dump.  Expected: there shouldn't be any gocb related goroutines running.")
	}

}
