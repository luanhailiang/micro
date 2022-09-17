package codec

import (
	"encoding/json"

	"github.com/luanhailiang/micro.git/proto/rpcmsg"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func ToBuff(msg proto.Message, isJson bool) (*rpcmsg.BuffMessage, error) {
	var err error
	var data []byte
	if isJson {
		data, err = json.Marshal(msg)
	} else {
		data, err = proto.Marshal(msg)
	}
	if err != nil {
		return nil, err
	}
	name := msg.ProtoReflect().Descriptor().FullName()
	buf := &rpcmsg.BuffMessage{
		Name: string(name),
		Data: data,
		Json: isJson,
	}
	return buf, nil
}

func UnBuff(buf *rpcmsg.BuffMessage) (proto.Message, error) {
	fullName := protoreflect.FullName(buf.Name)
	msg, err := protoregistry.GlobalTypes.FindMessageByName(fullName)
	if err != nil {
		return nil, err
	}
	obj := msg.New().Interface()
	if buf.Json {
		err = json.Unmarshal(buf.GetData(), obj)
	} else {
		err = proto.Unmarshal(buf.GetData(), obj)
	}
	if err != nil {
		return nil, err
	}
	return obj, nil
}
