package nats_pub

import (
	"os"
	"sync"

	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
)

var _nc *nats.Conn
var once sync.Once

//getClient 获取链接
func getClient() *nats.Conn {
	once.Do(func() {
		url := nats.DefaultURL
		if os.Getenv("NATS_URL") != "" {
			url = os.Getenv("NATS_URL")
		}
		var err error
		_nc, err = nats.Connect(url)
		if err != nil {
			log.Fatal(err)
		}
	})
	return _nc
}
