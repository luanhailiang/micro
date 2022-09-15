package rds

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"

	"github.com/luanhailiang/micro.git/network/with_val"
)

var db *DBClient

// DBClient 数据接口
type DBClient struct {
	name   string
	addr   string
	pass   string
	db     int
	zone   bool
	client *redis.Client
}

func init() {
	db = &DBClient{}
	db.name = os.Getenv("SERVE_NAME")
	db.addr = os.Getenv("REDIS_ADDR")
	db.pass = os.Getenv("REDIS_PASS")
	db.db = 0
	if os.Getenv("REDIS_DB") != "" {
		db.db, _ = strconv.Atoi(os.Getenv("REDIS_DB"))
	}
	if os.Getenv("NO_SPACE") == "" {
		db.zone = true
	}
	db.client = redis.NewClient(&redis.Options{
		Addr:     db.addr,
		Password: db.pass, // no password set
		DB:       db.db,   // use default DB
	})

	logrus.Infof("redis connect addr:%s db:%d name:%s", db.addr, db.db, db.name)
}

func GetClient() *redis.Client {
	return db.client
}

func Key(ctx context.Context, names ...string) string {
	if !db.zone {
		keys := []string{db.name}
		keys = append(keys, names...)
		return strings.Join(keys, "_")
	}
	mate, ok := with_val.FromMateContext(ctx)
	if !ok {
		logrus.Errorf("keyCtx lost mate")
	}
	if len(mate.Space) != 4 {
		num, err := strconv.Atoi(mate.Space)
		if err == nil {
			mate.Space = fmt.Sprintf("%04d", num)
		}
	}
	keys := []string{db.name, mate.Space}
	keys = append(keys, names...)
	return strings.Join(keys, "_")
}
