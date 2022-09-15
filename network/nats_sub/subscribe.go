package nats_sub

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/luanhailiang/micro.git/network/with_val"
	"github.com/luanhailiang/micro.git/proto/rpcmsg"
)

// Command 所有命令制定当前结构体
type Command struct {
	lock    sync.RWMutex
	cmd     reflect.Value
	indexes map[string]*nats.Subscription
}

var m *Command
var once sync.Once

func funToSub(name string) (string, bool) {
	index := strings.Index(name, "__")
	queue := index != -1
	var arr []string
	if queue {
		arr = strings.Split(name, "__")
	} else {
		arr = strings.Split(name, "_")
	}
	return fmt.Sprintf("%s.%s", strings.ToLower(arr[0]), arr[1]), queue
}
func subToFun(name string, queue bool) string {
	var gap string
	if queue {
		gap = "__"
	} else {
		gap = "_"
	}
	arr := strings.Split(name, ".")
	return fmt.Sprintf("%s%s%s", strings.ToUpper(arr[0]), gap, arr[1])
}

// Register CLI commands
func Register(cmd interface{}) {
	once.Do(func() {
		m = &Command{}
		m.indexes = map[string]*nats.Subscription{}
	})
	m.cmd = reflect.ValueOf(cmd).Elem()
	nc := getClient()
	for i := 0; i < m.cmd.NumMethod(); i++ {
		method := m.cmd.Type().Method(i)
		name, queue := funToSub(method.Name)
		if queue {
			nc.QueueSubscribe(name, os.Getenv("SERVE_NAME"), Handler)
		} else {
			nc.Subscribe(name, Handler)
		}
		log.Info("Sub --> ", name)
	}
}

func getIndex(index string) *nats.Subscription {
	m.lock.RLock()
	sub := m.indexes[index]
	m.lock.RUnlock()
	return sub
}

func SubIndex(index string, cb nats.MsgHandler) error {
	//判断是否已经监听
	sub := getIndex(index)
	if sub != nil {
		return nil
	}
	nc := getClient()
	sub, err := nc.Subscribe(index, cb)
	if err != nil {
		return err
	}
	m.lock.Lock()
	defer m.lock.Unlock()
	m.indexes[index] = sub
	return nil
}

func RemIndex(index string) error {
	//判断是否已经监听
	sub := getIndex(index)
	if sub == nil {
		return nil
	}
	err := sub.Unsubscribe()
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.indexes, index)
	return err
}

func Handler(msg *nats.Msg) {
	if m == nil {
		log.Errorf("handler not regist")
		return
	}
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("nats Call:%v panic:%s", msg, err)
			debug.PrintStack()
		}
	}()
	callMsg := &rpcmsg.CallMessage{}
	proto.Unmarshal(msg.Data, callMsg)

	dst, err := with_val.UnBuff(callMsg.Buff)
	if err != nil {
		log.Errorf("Handler->with_val.UnBuff:%s %d %v %s", callMsg.Buff.Name, callMsg.Buff.Json, callMsg.Buff.Data, err.Error())
		return
	}

	fullName := protoreflect.FullName(msg.Subject)
	// mt, err := protoregistry.GlobalTypes.FindMessageByName(fullName)
	// if err != nil {
	// 	if err == protoregistry.NotFound {
	// 		log.Error(err)
	// 		return
	// 	}
	// 	log.Errorf("could not resolve %q: %v", fullName, err)
	// 	return
	// }

	// dst := mt.New().Interface()
	// proto.Unmarshal(callMsg.Buff.Data, dst)

	ctx := with_val.NewMateContext(context.Background(), callMsg.Mate)

	name := subToFun(string(fullName), msg.Sub.Queue != "")
	//根据消息名称查找处理函数

	fun := m.cmd.MethodByName(string(name))
	if !fun.IsValid() {
		log.Errorf("!fun.IsValid()->name:%s", name)
		return
	}

	log.Debugf("Sub => (%v) %s [%v]", callMsg.GetMate(), fullName, dst)
	//调用相关函数
	fun.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(dst)})
}
