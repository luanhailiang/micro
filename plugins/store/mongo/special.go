package mgo

import (
	"context"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/proto"
)

func GetOneM(ctx context.Context, table string, filter interface{}, obj interface{}) error {
	c := GetCollection(ctx, table)
	cursor := c.FindOne(ctx, filter)
	err := cursor.Err()
	if err != nil {
		return err
	}
	err = cursor.Decode(obj)
	logrus.Debugf("mgo == FindOne->table:%s filter:%v obj:%v err:%v", table, filter, obj, err)
	return err
}

func GetOneAndMod(ctx context.Context, table string, filter interface{}, obj interface{}) *mongo.SingleResult {
	c := GetCollection(ctx, table)
	opts := options.FindOneAndUpdate().SetUpsert(true)
	ret := c.FindOneAndUpdate(ctx, filter, obj, opts)
	logrus.Debugf("mgo == FindOneAndUpdate->table:%s filter:%v obj:%v ret:%v", table, filter, obj, ret)
	return ret
}
func ModOne(ctx context.Context, table string, filter interface{}, obj interface{}) (*mongo.UpdateResult, error) {
	c := GetCollection(ctx, table)
	opts := options.Update().SetUpsert(true)
	ret, err := c.UpdateOne(ctx, filter, obj, opts)
	logrus.Debugf("mgo == UpdateOne->table:%s filter:%v obj:%v ret:%v err:%v", table, filter, obj, ret, err)
	return ret, err
}

func ModOneOpt(ctx context.Context, table string, filter interface{}, obj interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	c := GetCollection(ctx, table)
	ret, err := c.UpdateOne(ctx, filter, obj, opts...)
	logrus.Debugf("mgo == UpdateOne->table:%s filter:%v obj:%v ret:%v err:%v", table, filter, obj, ret, err)
	return ret, err
}

func SetOneM(ctx context.Context, table string, filter interface{}, obj interface{}) (*mongo.UpdateResult, error) {
	update := bson.M{
		"$set": obj,
	}
	return ModOne(ctx, table, filter, update)
}

func IncOneM(ctx context.Context, table string, filter interface{}, obj interface{}) (*mongo.UpdateResult, error) {
	update := bson.M{
		"$inc": obj,
	}
	return ModOne(ctx, table, filter, update)
}
func PushOneM(ctx context.Context, table string, filter interface{}, obj interface{}) (*mongo.UpdateResult, error) {
	update := bson.M{
		"$push": obj,
	}
	return ModOne(ctx, table, filter, update)
}

func fixBuf(obj proto.Message) any {
	fields := obj.ProtoReflect().Descriptor().Fields()
	field := fields.ByName("_id")
	if field == nil {
		return obj
	}
	id := obj.ProtoReflect().Get(field).String()
	if id == "" {
		return obj
	}
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		logrus.Errorf("mgo == fixBuf->ObjectIDFromHex id:%s err:%s", id, err.Error())
		return obj
	}

	data, err := bson.Marshal(obj)
	if err != nil {
		logrus.Errorf("mgo == fixBuf->Marshal obj:%v err:%s", obj, err.Error())
		return obj
	}
	fixData := bson.M{}
	err = bson.Unmarshal(data, &fixData)
	if err != nil {
		logrus.Errorf("mgo == fixBuf->Unmarshal data:%v err:%s", data, err.Error())
		return obj
	}
	fixData["_id"] = oid
	return fixData
}

func AddOne(ctx context.Context, obj proto.Message) (*mongo.InsertOneResult, error) {
	fullName := obj.ProtoReflect().Descriptor().FullName()
	table := string(fullName.Name())
	c := GetCollection(ctx, table)

	fixData := fixBuf(obj)
	ret, err := c.InsertOne(ctx, fixData)
	logrus.Debugf("mgo == InsertOne->table:%s obj:%v ret:%v err:%v", table, fixData, ret, err)
	return ret, err
}

func GetAll(ctx context.Context, table string, filter interface{}, result interface{}, opts ...*options.FindOptions) error {
	c := GetCollection(ctx, table)
	cursor, err := c.Find(ctx, filter, opts...)
	if err != nil {
		logrus.Errorf("mgo == Find->table:%s  filter:%v list:%v err:%v", table, filter, result, err)
		return err
	}
	err = cursor.All(ctx, result)
	logrus.Debugf("mgo == Find->table:%s  filter:%v list:%v err:%v", table, filter, result, err)
	return err
}

func SetAll(ctx context.Context, table string, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	c := GetCollection(ctx, table)
	ret, err := c.UpdateMany(ctx, filter, update, opts...)
	logrus.Debugf("mgo == UpdateMany->table:%s  filter:%v update:%v ret:%v err:%v", table, filter, update, ret, err)
	return ret, err
}

func DelAll(ctx context.Context, table string, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {

	c := GetCollection(ctx, table)
	ret, err := c.DeleteMany(ctx, filter, opts...)
	logrus.Debugf("mgo == DeleteMany->table:%s  filter:%v ret:%v, err:%v", table, filter, ret, err)
	return ret, err
}

func AddAll(ctx context.Context, table string, datas []interface{}, opts ...*options.InsertManyOptions) (*mongo.InsertManyResult, error) {
	c := GetCollection(ctx, table)
	ret, err := c.InsertMany(ctx, datas, opts...)
	logrus.Debugf("mgo == InsertMany->table:%s  datas:%v ret:%v, err:%v", table, datas, ret, err)
	return ret, err
}

func AddAllBuf(ctx context.Context, table string, datas []proto.Message, opts ...*options.InsertManyOptions) (*mongo.InsertManyResult, error) {
	c := GetCollection(ctx, table)
	documents := []interface{}{}

	ret, err := c.InsertMany(ctx, documents, opts...)
	logrus.Debugf("mgo == InsertMany->table:%s  datas:%v ret:%v, err:%v", table, documents, ret, err)
	return ret, err
}

func BulkWrite(ctx context.Context, table string, models []mongo.WriteModel, opts ...*options.BulkWriteOptions) (*mongo.BulkWriteResult, error) {
	c := GetCollection(ctx, table)
	ret, err := c.BulkWrite(ctx, models, opts...)
	logrus.Debugf("mgo == BulkWrite->table:%s  models:%v ret:%v, err:%v", table, models, ret, err)
	return ret, err
}

func ModAll(ctx context.Context, table string, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	c := GetCollection(ctx, table)
	ret, err := c.UpdateMany(ctx, filter, update, opts...)
	logrus.Debugf("mgo == UpdateMany->table:%s  filter:%v update:%v ret:%v err:%v", table, filter, update, ret, err)
	return ret, err
}

func SetAllM(ctx context.Context, table string, filter interface{}, obj interface{}) (*mongo.UpdateResult, error) {
	update := bson.M{
		"$set": obj,
	}
	return ModAll(ctx, table, filter, update)
}

func IncAllM(ctx context.Context, table string, filter interface{}, obj interface{}) (*mongo.UpdateResult, error) {
	update := bson.M{
		"$inc": obj,
	}
	return ModAll(ctx, table, filter, update)
}
func PushAllM(ctx context.Context, table string, filter interface{}, obj interface{}) (*mongo.UpdateResult, error) {
	update := bson.M{
		"$push": obj,
	}
	return ModAll(ctx, table, filter, update)
}
