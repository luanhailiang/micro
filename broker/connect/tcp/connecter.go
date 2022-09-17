package main

import (
	"bytes"
	"encoding/binary"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	"github.com/luanhailiang/micro/broker/register"
	"github.com/luanhailiang/micro/proto/broker"
	"github.com/luanhailiang/micro/proto/rpcmsg"
)

// NewConnecter 获取一个消息处理对象
func NewConnecter(conn net.Conn) *Connecter {
	c := Connecter{}
	c.Init(conn)
	c.attr = map[string]any{}
	return &c
}

// Connecter 网络消息处理模块
type Connecter struct {
	conn      net.Conn
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

// Init 初始化网络消息
func (c *Connecter) Init(conn net.Conn) error {
	c.conn = conn
	c.loginTime = time.Now().Unix()
	return nil
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
	bs, err := proto.Marshal(rpcBack)
	if err != nil {
		log.Errorf("proto.Marshal->err:%s", err)
		return
	}
	//判断是否压缩
	var id byte
	var cm byte
	if len(bs) > 1024 {
		// cm = 1
		// bs = DoZlibCompress(bs)
	}
	msg := bytes.NewBuffer([]byte{})
	msg.WriteByte(id)
	msg.WriteByte(cm)
	WriteUint32(msg, uint32(len(bs)))
	msg.Write(bs)
	n, err := c.conn.Write(msg.Bytes())
	if err != nil {
		log.Errorf("tcp Write err:%s", err.Error())
	}
	if n != msg.Len() {
		log.Errorf("tcp Write len:%d != %d", n, msg.Len())
	}
	//write 内部有锁，但是可能系统buff满，写不完，需要外部枷锁，或起线程 循环写入，压测时优化
	//没有超大的文件写入，只有短命令，这都能写满buff，网络极差，或者客户端断线，可以考虑断开
}

// IOLoop 接口循环
func (c *Connecter) IOLoop() {
	var l int
	var err error
	var buff bytes.Buffer
	var buffLen uint32

	tmp := make([]byte, 1024)

	for {
		//客户但长时间不发送数据断开
		if err := c.conn.SetReadDeadline(time.Now().Add(180 * time.Second)); err != nil {
			break
		}
		l, err = (c.conn).Read(tmp)
		//读取socket错误断开连接
		if err != nil {
			log.Infof("network break:%s", c.GetAddr())
			break
		}
		_, err = buff.Write(tmp[0:l])
		if err != nil {
			log.Errorf("%s buff.Write->err:%s", c.GetAddr(), err.Error())
			break
		}
		for {
			if buff.Len() < 6 {
				break
			}
			buf := bytes.NewBuffer(buff.Bytes()[2:6])
			err = binary.Read(buf, binary.LittleEndian, &buffLen)
			if err != nil {
				log.Errorf("%s binary.Read->err:%s", c.GetAddr(), err.Error())
				goto END //读取长度等能错下面的就别读了断开连接
			}
			if buff.Len() < int(buffLen+6) {
				break
			}
			buff.Next(1)       //丢掉id
			cm := buff.Next(1) //丢掉是否压缩
			buff.Next(4)       //丢掉读取的长度

			MsgData := buff.Next(int(buffLen))
			if cm[0] == 1 {
				MsgData = DoZlibUnCompress(MsgData)
			}

			buffMsg := &rpcmsg.BuffMessage{}
			err = proto.Unmarshal(MsgData, buffMsg)
			if err != nil {
				log.Errorf("proto.Unmarshal->err:%s", err)
				continue
			}

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
END: //结束释放资源
	c.Release()
}
