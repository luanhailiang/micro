package mgo

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/luanhailiang/micro.git/network/with_val"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var db *DBClient

// DBClient 数据接口
type DBClient struct {
	uri      string
	name     string
	zone     bool
	client   *mongo.Client
	database *mongo.Database
}

func init() {
	db = &DBClient{}
	db.uri = os.Getenv("MONGO_URI")
	db.name = os.Getenv("SERVE_NAME")
	if os.Getenv("NO_SPACE") == "" {
		db.zone = true
	}
	// Set client options
	clientOptions := options.Client().ApplyURI(db.uri)
	// Connect to MongoDB
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		logrus.Fatal(err)
	}
	// Check the connection
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		logrus.Fatal(err)
	}
	db.client = client
	db.database = client.Database(db.name)
	logrus.Infof("Connected to MongoDB! name:%s uri:%s ", db.name, db.uri)
}

// getDBClient 获取数据库链接
func GetDatabase() *mongo.Database {
	return db.database
}

// GetTable 获取数据表链接
func GetCollection(ctx context.Context, table string) *mongo.Collection {
	if !db.zone {
		return db.database.Collection(table)
	}
	mate, ok := with_val.FromMateContext(ctx)
	if !ok {
		logrus.Panic("keyCtx lost mate")
	}
	// if mate.Space == "" {
	// 	return db.database.Collection(table)
	// }
	if len(mate.Space) != 4 {
		num, err := strconv.Atoi(mate.Space)
		if err == nil {
			mate.Space = fmt.Sprintf("%04d", num)
		}
	}
	table = fmt.Sprintf("%s%s", table, mate.Space)
	return db.database.Collection(table)
}
