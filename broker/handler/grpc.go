package handler

import (
	"os"
	"strings"
	"time"

	"github.com/luanhailiang/micro/broker/manager"
	"github.com/luanhailiang/micro/broker/register"
	"github.com/luanhailiang/micro/plugins/codec"
	"github.com/luanhailiang/micro/plugins/events/nats_pub"
	"github.com/luanhailiang/micro/plugins/matedata"
	"github.com/luanhailiang/micro/proto/broker"
	"github.com/luanhailiang/micro/proto/events"
	"github.com/luanhailiang/micro/proto/rpcmsg"
)

func init() {
	m := &GrpcHandler{}
	if os.Getenv("USE_JSON") != "" {
		m.json = true
	}
	register.Register(&m)
}

type GrpcHandler struct {
	json bool
}

// 登录解析信息并加入频道
func (p *GrpcHandler) CmdLogin(c manager.Connecter, cmd *broker.CmdLogin) (*broker.CmdLogin, uint32, string) {
	mate, err := matedata.JwtMate(cmd.Token)
	if err != nil {
		return cmd, uint32(rpcmsg.Code_UNKNOWN), err.Error()
	}

	addr := c.GetAddr()
	index := strings.Index(addr, ":")
	if index != -1 {
		addr = addr[0:index]
	}
	mate.Clnip = addr

	preConn := manager.GetMasterInstance().Get(mate.Index)
	if preConn != nil {
		buff, _ := codec.ToBuff(&broker.CmdBreak{Type: 1}, p.json)
		back := &rpcmsg.BackMessage{
			Buff: buff,
		}
		manager.GetMasterInstance().Dis(preConn)
		preConn.Back(back)
		preConn.Close()

		now := time.Now().Unix()
		event := &events.Logout{
			Time: now,
			Role: mate.Index,
			Zone: mate.Space,
			Addr: c.GetAddr(),
			Plat: cmd.Plat,
			Last: now - c.GetLoginTime(),
			Type: 1,
		}
		nats_pub.PubMate(mate, event)
	}

	c.SetMate(mate)
	c.SetAttr("plat", cmd.Plat)
	//注册角色频道
	if mate.Space != "" {
		manager.GetMasterInstance().Add(c, broker.Topic_TopicServer, mate.Space)
	}
	if mate.Index != "" {
		manager.GetMasterInstance().Add(c, broker.Topic_TopicPlayer, mate.Index)
	}
	manager.GetMasterInstance().AddPlat(broker.Platform(cmd.Plat))
	event := &events.SignIn{
		Time: time.Now().Unix(),
		Role: mate.Index,
		Zone: mate.Space,
		Addr: c.GetAddr(),
		Plat: cmd.Plat,
	}
	nats_pub.PubMate(mate, event)
	return cmd, uint32(rpcmsg.Code_OK), ""
}

// 中途加入团体
func (p *GrpcHandler) CmdEnter(c manager.Connecter, cmd *broker.CmdEnter) *broker.CmdEnter {
	manager.GetMasterInstance().Add(c, cmd.Channel.Topic, cmd.Channel.Index)
	return cmd
}

// 中途离开频道
func (p *GrpcHandler) CmdLeave(c manager.Connecter, cmd *broker.CmdLeave) *broker.CmdLeave {
	manager.GetMasterInstance().Del(c, cmd.Channel.Topic, cmd.Channel.Index)
	return cmd
}

// 中途离开频道
func (p *GrpcHandler) CmdHeart(c manager.Connecter, cmd *broker.CmdHeart) *broker.CmdHeart {
	mate := c.GetMate()
	plat := c.GetAttr("plat").(int32)
	now := time.Now().Unix()
	event := &events.Heart{
		Time: now,
		Role: mate.Index,
		Zone: mate.Space,
		Addr: c.GetAddr(),
		Plat: plat,
		Last: now - c.GetLoginTime(),
	}
	nats_pub.PubMate(mate, event)
	return cmd
}

// 中途离开频道
func (p *GrpcHandler) CmdBreak(c manager.Connecter, cmd *broker.CmdBreak) {
	plat := c.GetAttr("plat").(int32)
	manager.GetMasterInstance().AddPlat(broker.Platform(plat))
	mate := c.GetMate()
	// fmt.Println(c, cmd)
	manager.GetMasterInstance().Dis(c)
	now := time.Now().Unix()
	event := &events.Logout{
		Time: now,
		Role: mate.Index,
		Zone: mate.Space,
		Addr: c.GetAddr(),
		Plat: plat,
		Last: now - c.GetLoginTime(),
	}
	nats_pub.PubMate(mate, event)
	nats_pub.SpacePubMate(mate, event)
}
