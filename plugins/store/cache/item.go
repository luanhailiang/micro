package mem

import (
	"time"

	"google.golang.org/protobuf/proto"
)

type Item struct {
	Name string
	Data proto.Message
	Time int64
}

func (p *Item) Load(msg proto.Message) {
	proto.Merge(msg, p.Data)
}

func (p *Item) Dump(msg proto.Message) {
	p.Data = proto.Clone(msg)
	p.Name = string(msg.ProtoReflect().Descriptor().FullName())
	p.Time = time.Now().Unix()
}

func (p *Item) TimeOut(num int64) bool {
	return p.Time+num < time.Now().Unix()
}
