package main

import (
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"
)

//StartServe 在main函数调用
func StartServe() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 7070))
	if err != nil {
		log.Fatalf("net.Listen->err:%s", err)
	}
	log.Infof("Tcp serve on port:%v", lis.Addr())

	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Errorf("ln.Accept->err:%s", err)
			continue
		}
		c := NewConnecter(conn)
		go c.IOLoop()
	}
}
