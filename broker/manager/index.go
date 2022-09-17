package manager

import (
	"sync"

	"github.com/luanhailiang/micro/proto/rpcmsg"
)

// Connecter 网络消息处理模块
type Connecter interface {
	Back(*rpcmsg.BackMessage)
	Close()
	GetAddr() string
	GetLoginTime() int64
	SetMate(*rpcmsg.MateMessage)
	GetMate() *rpcmsg.MateMessage
	SetAttr(string, any)
	GetAttr(string) any
}

type Index struct {
	lock  *sync.RWMutex
	index string
	conns map[Connecter]bool
}

// NewIndex 创建master
func NewIndex(index string) *Index {
	p := &Index{}
	p.lock = new(sync.RWMutex)
	p.index = index
	p.conns = map[Connecter]bool{}
	return p
}

func (p *Index) Len() int {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return len(p.conns)
}

func (p *Index) Get() Connecter {
	p.lock.Lock()
	defer p.lock.Unlock()
	for c := range p.conns {
		return c
	}
	return nil
}

func (p *Index) Add(conn Connecter) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.conns[conn] = true
}

func (p *Index) Del(conn Connecter) {
	p.lock.Lock()
	defer p.lock.Unlock()
	delete(p.conns, conn)
}

func (p *Index) Tel(msg *rpcmsg.BackMessage) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	for k := range p.conns {
		k.Back(msg)
	}
}
