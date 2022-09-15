package main

import (
	"os"

	"net/http"
	_ "net/http/pprof"

	"github.com/sirupsen/logrus"
)

func main() {
	if os.Getenv("DEBUG_LOG") != "" {
		logrus.SetLevel(logrus.DebugLevel)
	}
	go func() {
		logrus.Info(http.ListenAndServe(":6060", nil))
	}()

	StartServe()
}
