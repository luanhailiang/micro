package mgo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func AutoID(ctx context.Context, table string, space uint32) (int32, error) {
	c := GetCollection(ctx, table)
	opts := options.FindOneAndUpdate().SetUpsert(true)
	ret := c.FindOneAndUpdate(
		ctx,
		bson.M{"space": space},
		bson.M{"$inc": bson.M{"sequence_value": 1}},
		opts,
	)
	if ret.Err() != nil {
		if ret.Err() == mongo.ErrNoDocuments {
			return 0, nil
		}
		return 0, ret.Err()
	}
	raw, err := ret.DecodeBytes()
	if err != nil {
		return 0, err
	}
	count := raw.Lookup("sequence_value").AsInt32()
	return count, nil
}
