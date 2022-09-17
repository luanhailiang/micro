package manager

import (
	"sync"

	"github.com/luanhailiang/micro.git/plugins/events/nats_sub"
	"github.com/luanhailiang/micro.git/proto/rpcmsg"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

type Topic struct {
	lock    *sync.RWMutex
	topic   string
	indexes map[string]*Index
}

// NewConnecterSet 创建master
func NewTopic(topic string) *Topic {
	channel := &Topic{}
	channel.lock = new(sync.RWMutex)
	channel.topic = topic
	channel.indexes = map[string]*Index{}
	return channel
}

func (p *Topic) getIndex(index string) *Index {
	p.lock.RLock()
	indexObj, ok := p.indexes[index]
	p.lock.RUnlock()
	if ok {
		return indexObj
	}
	p.lock.Lock()
	defer p.lock.Unlock()
	indexObj, ok = p.indexes[index]
	if ok {
		return indexObj
	}
	indexObj = NewIndex(index)
	p.indexes[index] = indexObj
	return indexObj
}

func (p *Topic) Get(index string) Connecter {
	indexObj := p.getIndex(index)
	return indexObj.Get()
}

func (p *Topic) Add(index string, conn Connecter) {
	indexObj := p.getIndex(index)
	indexObj.Add(conn)
	if indexObj.Len() == 1 {
		nats_sub.SubIndex(index, p.Nat)
		logrus.Debugf("tpc ++ %s %s", p.topic, index)
	}
}

func (p *Topic) Del(index string, conn Connecter) {
	indexObj := p.getIndex(index)
	indexObj.Del(conn)
	//空的时候删除
	if indexObj.Len() > 0 {
		return
	}
	p.lock.Lock()
	defer p.lock.Unlock()
	delete(p.indexes, index)

	nats_sub.RemIndex(index)
	logrus.Debugf("topic -- %s %s", p.topic, index)
}

func (p *Topic) Tel(index string, msg *rpcmsg.BackMessage) {
	p.lock.RLock()
	indexObj, ok := p.indexes[index]
	p.lock.RUnlock()
	if !ok {
		return
	}
	indexObj.Tel(msg)
}

func (p *Topic) Nat(msg *nats.Msg) {
	index := msg.Subject
	cmd := &rpcmsg.BackMessage{}
	proto.Unmarshal(msg.Data, cmd)
	p.Tel(index, cmd)
	logrus.Debugf("tpc == %s %s %v", p.topic, index, cmd)
}
