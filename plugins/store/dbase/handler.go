package dbs

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/protobuf/proto"
	"gopkg.in/mgo.v2/bson"

	mem "github.com/luanhailiang/micro/plugins/store/cache"
	mgo "github.com/luanhailiang/micro/plugins/store/mongo"
	rds "github.com/luanhailiang/micro/plugins/store/redis"
)

var LOCAL_CACHE bool

func init() {
	if os.Getenv("LOCAL_CACHE") != "" {
		LOCAL_CACHE = true
	}
}

func GetOneByID(ctx context.Context, id string, obj proto.Message) (err error) {

	if LOCAL_CACHE { //如果开启内存缓存，针对信息类，如果排行，玩家信息,不可信与其他线程可能不同
		err = mem.GetManager().GetOne(ctx, id, obj)
		if err == nil {
			// logrus.Debugf("mem get one %s %v", index, obj)
			return err
		}
	}
	err = rds.GetOne(ctx, id, obj)
	if err == nil {
		if LOCAL_CACHE { //如果开启内存缓存，针对信息类，如果排行，玩家信息,不可信与其他线程可能不同
			err = mem.GetManager().SetOne(ctx, id, obj)
			if err != nil {
				return err
			}
		}
		// logrus.Debugf("rds get one %s %v", index, obj)
		return err
	}
	err = mgo.GetOneByID(ctx, id, obj)
	if err != nil {
		return err
	}
	if LOCAL_CACHE { //如果开启内存缓存，针对信息类，如果排行，玩家信息,不可信与其他线程可能不同
		err = mem.GetManager().SetOne(ctx, id, obj)
		if err != nil {
			return err
		}
	}
	err = rds.SetOne(ctx, id, obj)
	// logrus.Debugf("mgo get one %s %v", index, obj)
	return err
}

func SetOneByID(ctx context.Context, id string, obj proto.Message) (err error) {
	_, err = mgo.SetOneByID(ctx, id, obj)
	if err != nil {
		return err
	}
	err = mgo.GetOneByID(ctx, id, obj)
	if err != nil {
		return err
	}

	err = rds.SetOne(ctx, id, obj)
	if err != nil {
		return err
	}
	if LOCAL_CACHE { //如果开启内存缓存，针对信息类，如果排行，玩家信息
		err = mem.GetManager().SetOne(ctx, id, obj)
		if err != nil {
			return err
		}
	}
	return nil
}

func DelOneByID(ctx context.Context, id string, obj proto.Message) (err error) {
	_, err = mgo.DelOneByID(ctx, id, obj)
	if err != nil {
		return err
	}

	err = rds.DelOne(ctx, id, obj)
	if err != nil {
		return err
	}
	if LOCAL_CACHE { //如果开启内存缓存，针对信息类，如果排行，玩家信息
		mem.GetManager().DelOne(ctx, id, obj)
	}
	return err
}

func GetAllByID(ctx context.Context, datas map[string]proto.Message) error {
	for id, msg := range datas {
		err := GetOneByID(ctx, id, msg)
		if err != nil {
			return err
		}
	}
	return nil
}

// 传入bson.D必须按顺序，否则redis-key不同
func toIndex(filter bson.D) string {
	args := []string{}
	for _, e := range filter {
		var val string
		switch e.Value.(type) {
		case string:
			val = e.Value.(string)
		case primitive.ObjectID:
			val = e.Value.(primitive.ObjectID).Hex()
		default:
			val = fmt.Sprintf("%v", e.Value)
		}
		args = append(args, val)
	}
	index := strings.Join(args, ":")
	return index
}

func GetOne(ctx context.Context, filter bson.D, obj proto.Message) (err error) {
	index := toIndex(filter)
	if LOCAL_CACHE { //如果开启内存缓存，针对信息类，如果排行，玩家信息,不可信与其他线程可能不同
		err = mem.GetManager().GetOne(ctx, index, obj)
		if err == nil {
			// logrus.Debugf("mem get one %s %v", index, obj)
			return err
		}
	}
	err = rds.GetOne(ctx, index, obj)
	if err == nil {
		if LOCAL_CACHE { //如果开启内存缓存，针对信息类，如果排行，玩家信息,不可信与其他线程可能不同
			err = mem.GetManager().SetOne(ctx, index, obj)
			if err != nil {
				return err
			}
		}
		// logrus.Debugf("rds get one %s %v", index, obj)
		return err
	}
	err = mgo.GetOne(ctx, filter, obj)
	if err != nil {
		return err
	}
	if LOCAL_CACHE { //如果开启内存缓存，针对信息类，如果排行，玩家信息,不可信与其他线程可能不同
		err = mem.GetManager().SetOne(ctx, index, obj)
		if err != nil {
			return err
		}
	}
	err = rds.SetOne(ctx, index, obj)
	// logrus.Debugf("mgo get one %s %v", index, obj)
	return err
}

func SetOne(ctx context.Context, filter bson.D, obj proto.Message) (err error) {
	_, err = mgo.SetOne(ctx, filter, obj)
	if err != nil {
		return err
	}
	err = mgo.GetOne(ctx, filter, obj)
	if err != nil {
		return err
	}

	index := toIndex(filter)
	err = rds.SetOne(ctx, index, obj)
	if err != nil {
		return err
	}
	if LOCAL_CACHE { //如果开启内存缓存，针对信息类，如果排行，玩家信息
		err = mem.GetManager().SetOne(ctx, index, obj)
		if err != nil {
			return err
		}
	}
	return nil
}

func DelOne(ctx context.Context, filter bson.D, obj proto.Message) (err error) {
	_, err = mgo.DelOne(ctx, filter, obj)
	if err != nil {
		return err
	}
	index := toIndex(filter)
	err = rds.DelOne(ctx, index, obj)
	if err != nil {
		return err
	}
	if LOCAL_CACHE { //如果开启内存缓存，针对信息类，如果排行，玩家信息
		mem.GetManager().DelOne(ctx, index, obj)
	}
	return err
}
