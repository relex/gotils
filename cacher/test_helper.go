// Copyright 2021 RELEX Oy
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cacher

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
)

const (
	// Addr is the server address
	Addr     string = "localhost:12345"
	cacheDir string = "test_cache"
)

var (
	server http.Server
)

// StartHTTPServer starts a HTTP server in background
func StartHTTPServer(testFilePath string) {
	data := readFile(testFilePath)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host == Addr {
			w.Write(data)
		} else {
			w.Write([]byte(fmt.Sprintf("Unknown host: %s", r.Host)))
			return
		}
	})
	log.Printf("Now listening on %s...\n", Addr)
	lsnr, lsnrErr := net.Listen("tcp", Addr)
	if lsnrErr != nil {
		log.Panicf("cannot listen on %s: %s", Addr, lsnrErr)
	}
	server = http.Server{Handler: handler, Addr: Addr}
	go server.Serve(lsnr)
}

// StopHTTPServer stops a HTTP server
func StopHTTPServer() {
	server.Close()
	server = http.Server{}
	totalCacheRequests.Reset()
	totalRequests.Reset()
	failedRequests.Reset()
}

func readFile(path string) []byte {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Panicf("cannot read file: %s", path)
	}
	return data
}
