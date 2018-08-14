package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	timeout = time.Minute
)

func main() {
	httpClient := http.DefaultClient
	repliesMap := newRepliesMap()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s := strings.Split(r.URL.Path, "/")
		if len(s) != 2 {
			w.WriteHeader(404)
			return
		}
		channel := s[1]
		if channel == "" {
			fmt.Printf("response headers: %v", r.Header)
			correlationID := r.Header.Get("knative-correlation-id")
			replies := repliesMap.Get(correlationID)
			if replies == nil {
				// TODO: unknown correlationID; could be a timeout
			}
			reply, err := ioutil.ReadAll(r.Body)
			if err != nil {
				// TODO
			}
			fmt.Printf("REPLY %s: %s\n", correlationID, reply)
			replies <- string(reply)
		} else {
			fmt.Printf("SENDING TO %s\n", channel)
			// TODO: namespace
			url := fmt.Sprintf("http://%s-channel.default.svc.cluster.local", channel)
			req, err := http.NewRequest(http.MethodPost, url, r.Body)
			if err != nil {
				fmt.Fprintf(w, "Unable to create request %v", err)
				return
			}
			uuid, err := uuid.NewRandom()
			// TODO err
			correlationID := uuid.String()
			replyChan := make(chan string)
			repliesMap.Put(correlationID, replyChan)
			defer repliesMap.Delete(correlationID)
			req.Header = r.Header
			req.Header.Add("knative-correlation-id", correlationID)
			fmt.Printf("headers: %v\n", req.Header)
			res, err := httpClient.Do(req)
			if err != nil {
				fmt.Fprintf(w, "failed to make request %v", err)
				return
			} else if res.StatusCode >= 400 {
				fmt.Fprintf(w, "status: %d", res.StatusCode)
				return
			} else {
				select {
				case reply := <-replyChan:
					w.Write([]byte(reply))
				case <-time.After(timeout):
					w.WriteHeader(404)
				}
			}
			return
		}
	})
	http.ListenAndServe(":8080", nil)
}

// Type repliesMap implements a concurrent safe map of channels to send replies to, keyed by correlationIds
type repliesMap struct {
	m    map[string]chan<- string
	lock sync.RWMutex
}

func (replies *repliesMap) Delete(key string) {
	replies.lock.Lock()
	defer replies.lock.Unlock()
	delete(replies.m, key)
}

func (replies *repliesMap) Get(key string) chan<- string {
	replies.lock.RLock()
	defer replies.lock.RUnlock()
	return replies.m[key]
}

func (replies *repliesMap) Put(key string, value chan<- string) {
	replies.lock.Lock()
	defer replies.lock.Unlock()
	replies.m[key] = value
}

func newRepliesMap() *repliesMap {
	return &repliesMap{make(map[string]chan<- string), sync.RWMutex{}}
}
