package manager

import (
	"sync"

	"github.com/luanhailiang/micro.git/proto/broker"
)

type Cache struct {
	lock *sync.RWMutex
	keys map[Connecter]*[broker.Topic_TopicNumber]string
}

func NewCache() *Cache {
	index := &Cache{}
	index.lock = new(sync.RWMutex)
	index.keys = map[Connecter]*[broker.Topic_TopicNumber]string{}
	return index
}

func (p *Cache) Add(conn Connecter, topic broker.Topic, index string) (pre string) {
	p.lock.Lock()
	defer p.lock.Unlock()
	cache, ok := p.keys[conn]
	if !ok {
		cache = &[broker.Topic_TopicNumber]string{}
		p.keys[conn] = cache
	}
	if cache[topic] != "" {
		pre = cache[topic]
	}
	cache[topic] = index
	// fmt.Println(cache, p.keys[conn])
	return pre
}

func (p *Cache) Del(conn Connecter, topic broker.Topic) {
	p.lock.Lock()
	defer p.lock.Unlock()
	cache, ok := p.keys[conn]
	if !ok {
		return
	}
	cache[topic] = ""
}

func (p *Cache) Get(conn Connecter) *[broker.Topic_TopicNumber]string {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.keys[conn]
}

func (p *Cache) Dis(conn Connecter) {
	p.lock.Lock()
	defer p.lock.Unlock()
	delete(p.keys, conn)
}
