package nats_pub

import (
	"context"
	"os"

	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	"github.com/luanhailiang/micro/plugins/codec"
	"github.com/luanhailiang/micro/plugins/matedata"
	"github.com/luanhailiang/micro/proto/rpcmsg"
)

var isJson = false

func init() {
	if os.Getenv("IS_JSON") == "true" {
		isJson = true
	}
}

// Pub 调用函数
func PubMate(mate *rpcmsg.MateMessage, msg proto.Message) {
	buff, _ := codec.ToBuff(msg, isJson)
	name := proto.MessageName(msg)
	natsMsg := &rpcmsg.CallMessage{
		Mate: mate,
		Buff: buff,
	}
	natsData, _ := proto.Marshal(natsMsg)
	log.Debugf("Pub => (%v) %s [%v]", mate, name, msg)
	getClient().Publish(string(name), natsData)
}

// Pub 调用函数
func Pub(ctx context.Context, msg proto.Message) {
	mate, ok := matedata.FromMateContext(ctx)
	if !ok {
		log.Errorf("nats pub lost mate data:%s", proto.MessageName(msg))
	}
	PubMate(mate, msg)
}

// Pub 调用函数
func SpacePubMate(mate *rpcmsg.MateMessage, msg proto.Message) {
	buff, _ := codec.ToBuff(msg, isJson)
	name := proto.MessageName(msg)
	natsMsg := &rpcmsg.CallMessage{
		Mate: mate,
		Buff: buff,
	}
	natsData, _ := proto.Marshal(natsMsg)
	log.Debugf("Pub => (%v) %s [%v]", mate, mate.Space+"#"+string(name), msg)
	getClient().Publish(mate.Space+"#"+string(name), natsData)
}

// Pub 调用函数
func SpacePub(ctx context.Context, msg proto.Message) {
	mate, ok := matedata.FromMateContext(ctx)
	if !ok {
		log.Errorf("nats pub lost mate data:%s", proto.MessageName(msg))
	}
	SpacePubMate(mate, msg)
}

// Pub 调用函数
func IndexPub(index string, msg *rpcmsg.BackMessage) {
	data, _ := proto.Marshal(msg)
	log.Debugf("Pub => index:%s [%v]", index, msg)
	getClient().Publish(index, data)
}

// Pub 调用函数
func IndexPubMessage(index string, msg proto.Message) {
	buff, _ := codec.ToBuff(msg, isJson)
	back := &rpcmsg.BackMessage{
		Buff: buff,
	}
	IndexPub(index, back)
}
