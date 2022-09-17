package codec

import (
	"encoding/binary"
	"strconv"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// 设置成零，按理不会跟mongo重复
var processUnique = [4]byte{0, 0, 0, 0}

func GetRoleID(space, count uint32) uint32 {
	return space*1000000 + count
}

func GetSpaceCount(id uint32) (uint32, uint32) {
	space := id / 1000000 // id >> 20
	index := id % 1000000 // id & (1<<21 - 1)
	return space, index
}

func GetIDByObjectID(id primitive.ObjectID) uint32 {
	space := binary.BigEndian.Uint32(id[0:4])
	count := binary.BigEndian.Uint32(id[8:12])
	return GetRoleID(space, count)
}

// 合服后用户唯一id(space 必须是区服id整数，为了兼容hero mysql中的id)
func GetObjectIDByID(id uint32) primitive.ObjectID {
	//TODO 修改成和服规则
	space, index := GetSpaceCount(id)

	var b [12]byte

	binary.BigEndian.PutUint32(b[0:4], uint32(space))
	copy(b[4:8], processUnique[:])
	binary.BigEndian.PutUint32(b[8:12], uint32(index))

	return b
}

func GetObjectIDByIndex(index string) primitive.ObjectID {
	oid, err := primitive.ObjectIDFromHex(index)
	if err == nil {
		return oid
	}
	id, err := strconv.ParseInt(index, 10, 64)
	if err != nil {
		logrus.Errorf("GetObjectIDByIndex Atoi index:%s err:%s", index, err)
		return primitive.NilObjectID
	}

	return GetObjectIDByID(uint32(id))
}

func GetObjectIDByProto(msg proto.Message) primitive.ObjectID {
	fields := msg.ProtoReflect().Descriptor().Fields()
	field := fields.ByName("_id")
	if field == nil {
		logrus.Errorf("GetObjectIDByIndex error proto:%v", msg)
		return primitive.NilObjectID
	}
	xid := msg.ProtoReflect().Get(field).String()
	oid := GetObjectIDByIndex(xid)
	if oid.IsZero() {
		return oid
	}
	msg.ProtoReflect().Set(field, protoreflect.ValueOf(oid.Hex()))
	return oid
}

type DataPBStruct interface {
	GetXId() string
}
