package mgo

import (
	"context"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func GetOneByID(ctx context.Context, id string, obj proto.Message) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	filter := bson.M{"_id": oid}
	fullName := obj.ProtoReflect().Descriptor().FullName()
	table := string(fullName.Name())
	return GetOneM(ctx, table, filter, obj)
}

func SetOneByID(ctx context.Context, id string, obj proto.Message) (*mongo.UpdateResult, error) {
	fullName := obj.ProtoReflect().Descriptor().FullName()
	table := string(fullName.Name())

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	filter := bson.M{"_id": oid}

	fields := obj.ProtoReflect().Descriptor().Fields()
	field := fields.ByName("_id")
	if field != nil {
		obj.ProtoReflect().Set(field, protoreflect.ValueOfString(""))
	}
	return SetOneM(ctx, table, filter, obj)
}

func DelOneByID(ctx context.Context, id string, obj proto.Message) (*mongo.DeleteResult, error) {

	fullName := obj.ProtoReflect().Descriptor().FullName()
	table := string(fullName.Name())

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	filter := bson.M{"_id": oid}

	c := GetCollection(ctx, table)
	ret, err := c.DeleteOne(ctx, filter)
	logrus.Debugf("mgo == DeleteOne->table:%s filter:%v obj:%v ret:%v err:%v", table, filter, obj, ret, err)
	return ret, err
}

func GetAllByID(ctx context.Context, table string, ids []string, results interface{}, opts ...*options.FindOptions) error {
	oids := []primitive.ObjectID{}
	for _, id := range ids {
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return err
		}
		oids = append(oids, oid)
	}
	filter := bson.M{"_id": bson.M{"$in": oids}}
	c := GetCollection(ctx, table)
	cursor, err := c.Find(ctx, filter, opts...)
	if err != nil {
		logrus.Errorf("mgo == Find->table:%s  filter:%v list:%v err:%v", table, filter, results, err)
		return err
	}
	err = cursor.All(ctx, results)
	logrus.Debugf("mgo == Find->table:%s  filter:%v list:%v err:%v", table, filter, results, err)
	return err
}

func GetOne(ctx context.Context, filter interface{}, obj proto.Message) error {
	fullName := obj.ProtoReflect().Descriptor().FullName()
	table := string(fullName.Name())
	return GetOneM(ctx, table, filter, obj)
}

func SetOne(ctx context.Context, filter interface{}, obj proto.Message) (*mongo.UpdateResult, error) {
	fullName := obj.ProtoReflect().Descriptor().FullName()
	table := string(fullName.Name())

	fields := obj.ProtoReflect().Descriptor().Fields()
	field := fields.ByName("_id")
	if field != nil {
		obj.ProtoReflect().Set(field, protoreflect.ValueOfString(""))
	}
	return SetOneM(ctx, table, filter, obj)
}

func DelOne(ctx context.Context, filter interface{}, obj proto.Message) (*mongo.DeleteResult, error) {

	fullName := obj.ProtoReflect().Descriptor().FullName()
	table := string(fullName.Name())

	c := GetCollection(ctx, table)
	ret, err := c.DeleteOne(ctx, filter)
	logrus.Debugf("mgo == DeleteOne->table:%s filter:%v obj:%v ret:%v err:%v", table, filter, obj, ret, err)
	return ret, err
}
