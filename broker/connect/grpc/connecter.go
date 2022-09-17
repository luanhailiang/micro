package main

import (
	"io"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/proto"

	"github.com/luanhailiang/micro/broker/register"
	"github.com/luanhailiang/micro/proto/broker"
	"github.com/luanhailiang/micro/proto/rpcmsg"
)

// NewConnecter 获取一个消息处理对象
func NewConnecter(stream rpcmsg.Connect_LinkServer) *Connecter {
	c := Connecter{}
	c.Init(stream)
	c.attr = map[string]any{}
	return &c
}

// Connecter 网络消息处理模块
type Connecter struct {
	stream    rpcmsg.Connect_LinkServer
	addr      string
	mate      *rpcmsg.MateMessage
	attr      map[string]any
	loginTime int64
}

// GetAddr 获取地址
func (c *Connecter) GetAddr() string {
	return c.addr
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
	return c.mate
}

// Recv 处理自己的message
func (c *Connecter) Back(msg *rpcmsg.BackMessage) {
	err := c.stream.Send(msg)
	if err != nil {
		log.Errorf("c.Back:%v,%s", c.mate, err.Error())
	}
}

// Init 初始化网络消息
func (c *Connecter) Init(stream rpcmsg.Connect_LinkServer) error {
	if p, ok := peer.FromContext(stream.Context()); ok {
		c.addr = p.Addr.String()
	} else {
		c.addr = ""
	}
	c.stream = stream
	c.loginTime = time.Now().Unix()
	return nil
}

// Release 释放链接
func (c *Connecter) Release() {
	mate := c.GetMate()
	//灭有登录
	if mate == nil {
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
}

// Close 关闭链接
func (c *Connecter) Close() {
	c.mate = nil
	c.stream.Context().Done()
}

// IOLoop 消息获取
func (c *Connecter) IOLoop() error {
	//先绑定游戏对象
	for {
		in, err := c.stream.Recv()
		if err == io.EOF {
			c.Release()
			return nil
		}
		if err != nil {
			log.Errorf("IOLoop:%s", err.Error())
			c.Release()
			return err
		}
		ret := register.ProtectHandle(c, in)
		if ret != nil {
			c.Back(ret)
		}
	}
}
