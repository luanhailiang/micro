package grpc_cli

import (
	"context"
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/luanhailiang/micro/proto/rpcmsg"
)

var m *Manager
var once sync.Once

// GetManagerInstance 获取管理器
func GetManagerInstance() *Manager {
	once.Do(func() {
		m = &Manager{}
		m.lock = new(sync.RWMutex)
		m.objs = make(map[string]*Connecter)
	})
	return m
}

type Connecter struct {
	Name   string
	conn   *grpc.ClientConn
	Client rpcmsg.CommandClient
}

// Manager 管理器对象
type Manager struct {
	lock *sync.RWMutex
	objs map[string]*Connecter
}

// UnaryClientInterceptor for passing incoming metadata to outgoing metadata
func UnaryClientInterceptor(
	ctx context.Context,
	method string,
	req, reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption) error {
	// Take the incoming metadata and transfer it to the outgoing metadata
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if mdOut, ok := metadata.FromOutgoingContext(ctx); ok {
			md = metadata.Join(md, mdOut)
		}
		ctx = metadata.NewOutgoingContext(ctx, md)
		// logrus.Debug("UnaryClientInterceptor:", md)
	}
	return invoker(ctx, method, req, reply, cc, opts...)
}

// GetConnecterByName 获取链接
func (m *Manager) GetClient(name string) *Connecter {
	m.lock.RLock()
	conn, ok := m.objs[name]
	m.lock.RUnlock()
	if ok {
		return conn
	}
	m.lock.Lock()
	defer m.lock.Unlock()

	conn = &Connecter{}
	conn.Name = name

	connTemp, err := grpc.Dial(
		fmt.Sprintf("%s:9090", name),
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(UnaryClientInterceptor),
	)
	if err != nil {
		log.Errorf("error dail %s  %s", name, err)
		return nil
	}
	conn.conn = connTemp
	conn.Client = rpcmsg.NewCommandClient(conn.conn)
	m.objs[name] = conn
	return conn
}
