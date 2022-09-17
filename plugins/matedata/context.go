package matedata

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc/metadata"

	"github.com/luanhailiang/micro/proto/rpcmsg"
)

type mateMessageKey struct{}

// NewContext returns a new Context that carries value u.
func NewMateContext(ctx context.Context, u *rpcmsg.MateMessage) context.Context {
	// return context.WithValue(ctx, mateMessageKey{}, u)
	if u == nil {
		return ctx
	}
	data, err := json.Marshal(u)
	if err != nil {
		logrus.Error("NewMateContext", err.Error())
		return ctx
	}
	md := metadata.Pairs("meta", string(data))
	if mdIncoming, ok := metadata.FromIncomingContext(ctx); ok {
		md = metadata.Join(md, mdIncoming)
	}
	return metadata.NewIncomingContext(ctx, md)
}

// FromContext returns the User value stored in ctx, if any.
func FromMateContext(ctx context.Context) (*rpcmsg.MateMessage, bool) {
	// u, ok := ctx.Value(mateMessageKey{}).(*rpcmsg.MateMessage)
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, false
	}
	t, ok := md["meta"]
	if !ok || len(t) == 0 {
		return nil, false
	}
	meta := &rpcmsg.MateMessage{}
	err := json.Unmarshal([]byte(t[0]), meta)
	if err != nil {
		logrus.Error("FromMateContext", err.Error())
		return nil, false
	}
	return meta, ok
}

func RoleFilter(ctx context.Context) bson.D {
	mate, ok := FromMateContext(ctx)
	if !ok {
		return nil
	}
	oid, err := primitive.ObjectIDFromHex(mate.Index)
	if err != nil {
		logrus.Error("RoleFilter", err.Error())
		return nil
	}
	return bson.D{bson.E{Key: "_id", Value: oid}}
}

func AddFromContext(ctx context.Context, from string) context.Context {
	if from == "" {
		return ctx
	}
	md := metadata.Pairs("from", from)
	if mdIncoming, ok := metadata.FromIncomingContext(ctx); ok {
		md = metadata.Join(mdIncoming, md)
	}
	return metadata.NewIncomingContext(ctx, md)
}

func GetFromContext(ctx context.Context) []string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil
	}
	return md["from"]
}
