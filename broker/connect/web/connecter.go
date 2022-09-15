package main

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	"github.com/luanhailiang/micro.git/broker/register"
	"github.com/luanhailiang/micro.git/proto/broker"
	"github.com/luanhailiang/micro.git/proto/rpcmsg"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512 * 10
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024 * 10,
	WriteBufferSize: 1024 * 10,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// NewConnecter 获取一个消息处理对象
func NewConnecter(conn *websocket.Conn) *Connecter {
	c := Connecter{}
	c.conn = conn
	c.loginTime = time.Now().Unix()
	c.send = make(chan []byte, 256)
	c.attr = map[string]any{}
	return &c
}

// Connecter 网络消息处理模块
type Connecter struct {
	conn      *websocket.Conn
	send      chan []byte
	mate      *rpcmsg.MateMessage
	attr      map[string]any
	loginTime int64
}

// GetAddr 获取地址
func (c *Connecter) GetAddr() string {
	return c.conn.RemoteAddr().String()
}

// GetLoginTime 获取登录时间
func (c *Connecter) GetLoginTime() int64 {
	return c.loginTime
}

func (c *Connecter) SetMate(mate *rpcmsg.MateMessage) {
	c.mate = mate
}
func (c *Connecter) GetMate() *rpcmsg.MateMessage {
	return c.mate
}
func (c *Connecter) SetAttr(key string, val any) {
	c.attr[key] = val
}

func (c *Connecter) GetAttr(key string) any {
	return c.attr[key]
}

// Release 释放链接
func (c *Connecter) Release() {
	mate := c.GetMate()
	//灭有登录
	if mate == nil {
		c.conn.Close()
		return
	}
	dis := &broker.CmdBreak{}
	name := proto.MessageName(dis)
	data, _ := proto.Marshal(dis)
	buff := &rpcmsg.BuffMessage{
		Name: string(name),
		Data: data,
	}
	register.ProtectHandle(c, buff)

	c.conn.Close()
}

// Close 关闭链接
func (c *Connecter) Close() {
	c.mate = nil
	// c.conn.Close()
}

// Back 处理自己的message
func (c *Connecter) Back(rpcBack *rpcmsg.BackMessage) {
	log.Debug("Back:", rpcBack)
	if rpcBack.Info != "" {
		rpcBack.Info = strings.Replace(rpcBack.Info, "|", "#", -1)
	}
	if rpcBack.Buff == nil {
		log.Errorf("Back->error message:", rpcBack)
		return
	}
	data := fmt.Sprintf("%d|%s|%s|%s", rpcBack.Code, rpcBack.Info, rpcBack.Buff.Name, string(rpcBack.Buff.Data))
	c.send <- []byte(data)
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Connecter) readPump() {
	defer c.Release()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Errorf("ReadMessage error: %v", err)
			}
			break
		}
		log.Debugf("c.conn.ReadMessage:%s", string(message))
		datas := bytes.SplitN(message, []byte{'|'}, 2)
		if len(datas) != 2 {
			log.Errorf("error message format->err:%s", string(message))
			break
		}

		buffMsg := &rpcmsg.BuffMessage{}
		buffMsg.Name = string(datas[0])
		buffMsg.Data = datas[1]
		buffMsg.Json = true

		// log.Errorf("IOLoop->Buff:%v", Buff)
		rpcBack := register.ProtectHandle(c, buffMsg)
		if rpcBack == nil {
			continue
		}
		if rpcBack.Buff == nil && rpcBack.Code == 0 {
			continue
		}
		c.Back(rpcBack)
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Connecter) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			log.Debugf("c.conn.WriteMessage:%s", string(message))
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := NewConnecter(conn)

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}
