package handler

import (
	"context"
	"time"

	"github.com/luanhailiang/micro.git/broker/manager"
	"github.com/luanhailiang/micro.git/network/nats_pub"
	"github.com/luanhailiang/micro.git/network/nats_sub"
	"github.com/luanhailiang/micro.git/network/with_val"
	"github.com/luanhailiang/micro.git/proto/broker"
	"github.com/luanhailiang/micro.git/proto/events"
	"github.com/luanhailiang/micro.git/proto/rpcmsg"
)

func init() {
	m := &NatsHandler{}
	nats_sub.Register(&m)
}

type NatsHandler struct {
}

func (p *NatsHandler) BROKER_SubBroad(ctx context.Context, msg *broker.SubBroad) {
	if len(msg.Channel.GetArray()) != 0 {
		for _, index := range msg.Channel.GetArray() {
			manager.GetMasterInstance().Tel(msg.Back, msg.Channel.Topic, index)
		}
	}
	if msg.Channel.GetIndex() != "" {
		manager.GetMasterInstance().Tel(msg.Back, msg.Channel.Topic, msg.Channel.GetIndex())
	}
}

func (p *NatsHandler) BROKER_Beyond(ctx context.Context, msg *events.Beyond) {
	mate, _ := with_val.FromMateContext(ctx)

	preConn := manager.GetMasterInstance().Get(msg.Role)
	if preConn != nil {
		buff, _ := with_val.ToBuff(&broker.CmdBreak{Type: 2}, false)
		back := &rpcmsg.BackMessage{
			Buff: buff,
		}
		manager.GetMasterInstance().Dis(preConn)
		preConn.Back(back)
		preConn.Close()

		now := time.Now().Unix()
		event := &events.Logout{
			Time: now,
			Role: msg.Role,
			Zone: msg.Zone,
			Addr: msg.Addr,
			Plat: msg.Plat,
			Last: msg.Last,
			Type: 2,
		}
		nats_pub.PubMate(mate, event)
	}
}
