package grpc_svr

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sync"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/luanhailiang/micro/plugins/matedata"
	"github.com/luanhailiang/micro/proto/rpcmsg"
)

// Command 所有命令制定当前结构体
type Command struct {
	name string
	// pkg string
	cmd reflect.Value
}

var m *Command
var once sync.Once

// Register CLI commands
func Register(cmd interface{}) {
	once.Do(func() {
		m = &Command{}
		m.name = os.Getenv("SERVE_NAME")
	})
	m.cmd = reflect.ValueOf(cmd).Elem()

	for i := 0; i < m.cmd.NumMethod(); i++ {
		method := m.cmd.Type().Method(i)
		logrus.Info("Cmd --> ", method.Name)
	}
}

// Call 命令处理主入口
func Call(ctx context.Context, msg *rpcmsg.CallMessage) (*rpcmsg.BackMessage, error) {
	if m == nil {
		return nil, fmt.Errorf("handler not regist")
	}
	fullName := protoreflect.FullName(msg.Buff.Name)
	name := fullName.Name()
	mt, err := protoregistry.GlobalTypes.FindMessageByName(fullName)
	if err != nil {
		return nil, fmt.Errorf("could not resolve %q: %v", msg.Buff.Name, err)
	}
	dst := mt.New().Interface()
	//如果是json格式用json解析
	if msg.Buff.Json {
		json.Unmarshal(msg.Buff.GetData(), dst)
	} else {
		proto.Unmarshal(msg.Buff.GetData(), dst)
	}

	//根据消息名称查找处理函数
	fun := m.cmd.MethodByName(string(name))

	if !fun.IsValid() {
		return nil, fmt.Errorf("!fun.IsValid()->name:%s", name)
	}

	if msg.GetMate() != nil {
		ctx = matedata.NewMateContext(ctx, msg.GetMate())
	} else {
		msg.Mate, _ = matedata.FromMateContext(ctx)
	}

	logrus.Debugf("Cmd => (%v) %s [%v]", msg.Mate, name, dst)
	ret := fun.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(dst)})
	//调用相关函数

	// if len(ret) == 0 {
	// 	return nil, nil
	// }

	backMesasge := &rpcmsg.BackMessage{}
	//检测返回值，并推送到前段
	switch len(ret) {
	case 3:
		if info, ok := ret[2].Interface().(string); ok {
			if info != "" {
				info = m.name + "->" + info
			}
			backMesasge.Info = info
		}
		fallthrough
	case 2:
		if code, ok := ret[1].Interface().(uint32); ok {
			backMesasge.Code = code
		}
		fallthrough
	case 1:
		if ret[0].IsNil() {
			// if backMesasge.Code == 0 {
			// 	return nil, nil
			// }
			logrus.Debugf("Cmd <= (%v) %s [%v]", msg.GetMate(), name, backMesasge)
			return backMesasge, nil
		}
		retmsg := ret[0].Interface().(proto.Message)
		logrus.Debugf("Cmd <= (%v) %s [%v] %d %s", msg.GetMate(), name, retmsg, backMesasge.Code, backMesasge.Info)
		var data []byte
		//json格式来的就json格式返回去
		if msg.Buff.Json {
			data, err = json.Marshal(retmsg)
		} else {
			data, err = proto.Marshal(retmsg)
		}
		if err != nil {
			return nil, err
		}
		base := &rpcmsg.BuffMessage{
			Json: msg.Buff.Json,
			Name: string(retmsg.ProtoReflect().Descriptor().FullName()),
			Data: data,
		}
		backMesasge.Buff = base
	}
	return backMesasge, nil
}
