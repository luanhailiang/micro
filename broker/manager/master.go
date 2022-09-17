package manager

import (
	"hash/fnv"

	"github.com/luanhailiang/micro/proto/broker"
	"github.com/luanhailiang/micro/proto/rpcmsg"
	"github.com/sirupsen/logrus"
)

var master *Master

func init() {
	master = newMasterInstance()
}

const (
	hash_len = 256
)

func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32() % hash_len
}

// GetMasterInstance 获取管理器
func GetMasterInstance() *Master {
	return master
}

func newMasterInstance() *Master {
	m := &Master{}
	m.topics = [broker.Topic_TopicNumber][hash_len]*Topic{}
	for i := 0; i < int(broker.Topic_TopicNumber.Number()); i++ {
		for j := 0; j < hash_len; j++ {
			m.topics[i][j] = NewTopic(broker.Topic_name[int32(i)])
		}
	}

	m.caches = [hash_len]*Cache{}
	for i := 0; i < hash_len; i++ {
		m.caches[i] = NewCache()
	}
	m.counts = [broker.Platform_PlatformMax]int64{}
	return m
}

// Master 管理器对象
type Master struct {
	topics [broker.Topic_TopicNumber][hash_len]*Topic
	caches [hash_len]*Cache
	counts [broker.Platform_PlatformMax]int64
}

func (p *Master) AddPlat(plat broker.Platform) {
	p.counts[plat]++
}
func (p *Master) DelPlat(plat broker.Platform) {
	p.counts[plat]--
}

func (p *Master) getTopic(topic broker.Topic, index string) *Topic {
	i := hash(index)
	return p.topics[topic][i]
}

func (p *Master) getCache(conn Connecter) *Cache {
	i := hash(conn.GetAddr())
	return p.caches[i]
}

func (p *Master) Len() int {
	return len(p.caches)
}

func (p *Master) Get(index string) Connecter {
	if index == "" {
		return nil
	}
	topicObj := p.getTopic(broker.Topic_TopicPlayer, index)
	return topicObj.Get(index)
}

func (p *Master) Add(conn Connecter, topic broker.Topic, index string) {
	if index == "" {
		return
	}
	indexCache := p.getCache(conn)
	preIndex := indexCache.Add(conn, topic, index)

	topicObj := p.getTopic(topic, index)
	if preIndex != "" {
		topicObj.Del(preIndex, conn)
	}
	topicObj.Add(index, conn)
	logrus.Debugf("mgr ++ %s %s %s", conn.GetAddr(), topic, index)
}

func (p *Master) Del(conn Connecter, topic broker.Topic, index string) {
	indexCache := p.getCache(conn)
	indexCache.Del(conn, topic)

	topicObj := p.getTopic(topic, index)
	topicObj.Del(index, conn)
	logrus.Debugf("mgr -- %s %s %s", conn.GetAddr(), topic, index)
}

func (p *Master) Tel(msg *rpcmsg.BackMessage, topic broker.Topic, index string) {
	topicObj := p.getTopic(topic, index)
	topicObj.Tel(index, msg)
}

func (p *Master) Dis(conn Connecter) {
	indexCache := p.getCache(conn)
	keys := indexCache.Get(conn)
	// fmt.Println(conn, indexCache, keys)
	if keys == nil {
		logrus.Warnf("mgr ** lost %s", conn.GetAddr())
		return
	}
	for i := 0; i < len(keys); i += 1 {
		topic := broker.Topic(i)
		index := keys[i]
		if index == "" {
			continue
		}
		topicObj := p.getTopic(topic, index)
		topicObj.Del(index, conn)
		logrus.Debugf("mgr -- %s %s %s", conn.GetAddr(), topic, index)
	}
	indexCache.Dis(conn)
}
