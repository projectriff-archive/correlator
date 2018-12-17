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

var httpClient = http.DefaultClient
var replyChannels = newRepliesMap()
var claimChecks = make(map[string]string)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			handlePost(w, r)
		} else {
			handleGet(w, r)
		}
	})
	http.ListenAndServe(":8080", nil)
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	namespace, channel := parsePathToChannel(r.URL.Path)
	if channel == "" { // handle reply
		fmt.Printf("response headers: %v", r.Header)
		correlationID := r.Header.Get("knative-correlation-id")
		reply, err := ioutil.ReadAll(r.Body)
		if err != nil {
			// TODO
		}
		fmt.Printf("REPLY %s: %s\n", correlationID, reply)
		replyChannel := replyChannels.Get(correlationID)
		if replyChannel != nil {
			replyChannel <- string(reply)
		} else {
			// timeout or non-blocking request
			claimChecks[correlationID] = string(reply)
		}
	} else {
		uuid, err := uuid.NewRandom()
		if err != nil {
			fmt.Print("failed to create correlation ID")
			w.WriteHeader(500)
			return
		}
		correlationID := uuid.String()
		blocking := r.Header.Get("knative-blocking-request") == "true"
		var replyChan chan string
		if blocking {
			replyChan = make(chan string, 1)
			replyChannels.Put(correlationID, replyChan)
			defer replyChannels.Delete(correlationID)
		}
		err = sendToChannel(namespace, channel, correlationID, r)
		if err != nil {
			fmt.Fprintf(w, "Failed to send to channel %v", err)
			return
		}
		if !blocking {
			w.Header().Add("knative-correlation-id", correlationID)
			w.WriteHeader(202)
			return
		}
		select {
		case reply := <-replyChan:
			w.Write([]byte(reply))
		case <-time.After(timeout):
			w.WriteHeader(404)
		}
		return
	}
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	s := strings.Split(r.URL.Path, "/")
	if len(s) != 2 {
		w.WriteHeader(404)
		return
	}
	key := s[1]
	w.Write([]byte(claimChecks[key]))
}

func parsePathToChannel(path string) (namespace, channel string) {
	s := strings.Split(path, "/")
	if len(s) >= 3 {
		namespace = s[1]
		channel = s[2]
	}
	return
}

func sendToChannel(namespace, channel string, correlationID string, r *http.Request) error {
	fmt.Printf("SENDING TO %s/%s\n", namespace, channel)
	url := fmt.Sprintf("http://%s-channel.%s.svc.cluster.local", channel, namespace)
	req, err := http.NewRequest(http.MethodPost, url, r.Body)
	if err != nil {
		return err
	}
	req.Header = r.Header
	req.Header.Add("knative-correlation-id", correlationID)
	fmt.Printf("headers: %v\n", req.Header)
	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode >= 400 {
		return fmt.Errorf("status: %d", res.StatusCode)
	}
	return nil
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
