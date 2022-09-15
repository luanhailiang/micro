// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"

	_ "github.com/luanhailiang/micro.git/broker/handler"
)

var addr = flag.String("addr", ":8080", "http service address")

func main() {
	if os.Getenv("DEBUG_LOG") != "" {
		logrus.SetLevel(logrus.DebugLevel)
	}
	go func() {
		logrus.Info(http.ListenAndServe(":6060", nil))
	}()

	flag.Parse()
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(w, r)
	})
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
