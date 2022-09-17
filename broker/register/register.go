package register

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime/debug"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/luanhailiang/micro/broker/manager"
	"github.com/luanhailiang/micro/plugins/matedata"
	"github.com/luanhailiang/micro/plugins/message/grpc_cli"
	"github.com/luanhailiang/micro/proto/rpcmsg"
)

// Command 所有命令制定当前结构体
type Command struct {
	cmd reflect.Value
}

var m *Command
var once sync.Once

// Register CLI commands
func Register(cmd interface{}) {
	once.Do(func() {
		m = &Command{}
	})
	m.cmd = reflect.ValueOf(cmd).Elem()
	for i := 0; i < m.cmd.NumMethod(); i++ {
		method := m.cmd.Type().Method(i)
		log.Info("Cmd --> ", method.Name)
	}
}

// Call 命令处理主入口
func Call(c manager.Connecter, msg *rpcmsg.BuffMessage) (*rpcmsg.BackMessage, error) {
	if m == nil {
		return nil, fmt.Errorf("handler not regist")
	}
	fullName := protoreflect.FullName(msg.Name)
	name := fullName.Name()

	mt, err := protoregistry.GlobalTypes.FindMessageByName(fullName)
	if err != nil {
		if err == protoregistry.NotFound {
			return nil, err
		}
		return nil, fmt.Errorf("could not resolve %q: %v", msg.Name, err)
	}
	dst := mt.New().Interface()
	//如果是json格式用json解析
	if msg.Json {
		json.Unmarshal(msg.GetData(), dst)
	} else {
		proto.Unmarshal(msg.GetData(), dst)
	}

	log.Debugf("cmd => %s (%v) %s [%v]", c.GetAddr(), c.GetMate(), name, dst)
	//根据消息名称查找处理函数
	fun := m.cmd.MethodByName(string(name))

	if !fun.IsValid() {
		return nil, fmt.Errorf("!fun.IsValid()->name:%s", name)
	}

	ret := fun.Call([]reflect.Value{reflect.ValueOf(c), reflect.ValueOf(dst)})
	//调用相关函数

	if len(ret) == 0 {
		return nil, nil
	}
	backMesasge := &rpcmsg.BackMessage{}
	//检测返回值，并推送到前段
	switch len(ret) {
	case 3:
		if ret[2].Interface() != nil {
			if info, ok := ret[2].Interface().(string); ok {
				backMesasge.Info = info
			}
		}
		fallthrough
	case 2:
		if ret[1].Interface() != nil {
			if code, ok := ret[1].Interface().(uint32); ok {
				backMesasge.Code = code
			} else if code, ok := ret[1].Interface().(int32); ok {
				backMesasge.Code = uint32(code)
			} else if code, ok := ret[1].Interface().(int); ok {
				backMesasge.Code = uint32(code)
			}
		}
		fallthrough
	case 1:
		if ret[0].Interface() != nil {
			retmsg := ret[0].Interface().(proto.Message)
			log.Debugf("cmd <= %s (%v) %s [%v] {%d:%s}", c.GetAddr(), c.GetMate(), name, retmsg, backMesasge.Code, backMesasge.Info)
			var data []byte
			//json格式来的就json格式返回去
			if msg.Json {
				data, err = json.Marshal(retmsg)
			} else {
				data, err = proto.Marshal(retmsg)
			}
			if err != nil {
				return nil, err
			}
			base := &rpcmsg.BuffMessage{
				Json: msg.Json,
				Name: string(retmsg.ProtoReflect().Descriptor().FullName()),
				Data: data,
			}
			backMesasge.Buff = base
		}
	}
	return backMesasge, nil
}

func ProtectHandle(c manager.Connecter, cmd *rpcmsg.BuffMessage) *rpcmsg.BackMessage {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("handle cmd:%v panic:%s", cmd, err)
			debug.PrintStack()
		}
	}()
	pre := time.Now().UnixMilli()
	fullName := protoreflect.FullName(cmd.Name)
	parent := fullName.Parent()
	var ret *rpcmsg.BackMessage
	var err error
	//如果是网关命令本地调用不转发
	if string(parent) == "broker" {
		ret, err = Call(c, cmd)
	} else {
		mate := c.GetMate()
		if mate == nil {
			log.Errorf("handle not login:%s", c.GetAddr())
			return nil
		}
		ctx := matedata.NewMateContext(context.Background(), mate)
		ret, err = grpc_cli.CallBuff(ctx, cmd)
	}
	if err != nil {
		log.Errorf("handle cmd:%v err:%s", cmd, err.Error())
	}
	now := time.Now().UnixMilli()
	if now-pre > 10 {
		log.Errorf("handle cmd:%v use:%dms", cmd, now-pre)
	}
	return ret
}
