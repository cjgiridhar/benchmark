package main

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

type Request struct {
	count       int64
	concurrency int
	httpRequest *http.Request
	status      int64
	mutex       sync.Mutex
}

// Response contains an http.Response and an error if occured while making the request.
type Response struct {
	httpResponse *http.Response
	err          error
}

// HttpResponse is the getter method to get the http.Response of a concurrent Request.
func (res Response) HttpResponse() *http.Response {
	return res.httpResponse
}

// Error is the getter method to get the error of a concurrent Request.
func (res Response) Error() error {
	return res.err
}

// NewRequest is the constructor for concurrent.Request. It takes a defined http.Request,
// count which specifies the number of request to be made and concurrency of the requests.
func NewRequest(httpRequest *http.Request, count int64, concurrency int) (req *Request) {
	return &Request{
		count:       count,
		concurrency: concurrency,
		httpRequest: httpRequest,
		status:      0,
	}
}

// MakeSync makes the given requests in a blocking manner and returns when all the requests
// have been completed. It returns a channel of Responses corresponding to each request.
func (req *Request) MakeSync() (res chan Response) {
	res = make(chan Response, req.count)
	defer close(res)

	wg := sync.WaitGroup{}

	for i := 0; i < req.concurrency; i++ {
		wg.Add(1)
		go func() {
			for {
				req.mutex.Lock()
				if req.status >= req.count {
					req.mutex.Unlock()
					break
				}

				req.status++
				req.mutex.Unlock()
				newRes := Response{}
				newRes.httpResponse, newRes.err = http.DefaultClient.Do(req.httpRequest)
				res <- newRes
			}

			wg.Done()
			return
		}()
	}

	wg.Wait()

	return
}

func (req *Request) Status() (completed float32) {
	req.mutex.Lock()
	defer req.mutex.Unlock()
	//fmt.Println(req.status)
	return float32(req.status) / float32(req.count) * 100
}

func main() {
	url := os.Args[1]
	httpRequest, _ := http.NewRequest("GET", url, nil)

	// Parallelism of the request
	concurrency := 10

	// Total number of requests to be made.
	numberOfRequests := int64(100)

	concurrentRequest := NewRequest(httpRequest, numberOfRequests, concurrency)

	startTime := time.Now()
	go func() {
		concurrentRequest.MakeSync()
		completetionTime := time.Now().Sub(startTime)
		fmt.Printf("\n\nTime required to complete all requests: %v\n", completetionTime)
	}()

	var count int = 0
	tick := time.NewTicker(100 * time.Millisecond)
	for range tick.C {
		status := concurrentRequest.Status()
		timeElapsed := time.Now().Sub(startTime)
		if status == float32(numberOfRequests) {
			if count == 1 {
				os.Exit(0)
			}
			count += 1
		}
		fmt.Printf("\n%f requests sent, Time elapsed: %v", status, timeElapsed)
	}
}
