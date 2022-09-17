package grpc_cli

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/luanhailiang/micro/plugins/codec"
	"github.com/luanhailiang/micro/plugins/matedata"
	"github.com/luanhailiang/micro/proto/rpcmsg"
)

func Cmd(ctx context.Context, msg proto.Message) (uint32, string) {
	ctx = matedata.AddFromContext(ctx, os.Getenv("SERVE_NAME"))

	buff, err := codec.ToBuff(msg, false)
	if err != nil {
		return uint32(codes.Unknown), err.Error()
	}
	ret, err := CallBuff(ctx, buff)
	if err != nil {
		return uint32(codes.Unknown), err.Error()
	}
	if ret.Code == 0 {
		err = proto.Unmarshal(ret.Buff.Data, msg)
		if err != nil {
			logrus.Debugf("exe <= %s [%v] err:%v", ret.Buff.Name, msg, err)
		}
	}
	return ret.Code, ret.Info
}

func Call(ctx context.Context, msg proto.Message) (*rpcmsg.BackMessage, error) {
	ctx = matedata.AddFromContext(ctx, os.Getenv("SERVE_NAME"))

	buff, err := codec.ToBuff(msg, false)
	if err != nil {
		return nil, err
	}
	return CallBuff(ctx, buff)
}

func CallBuff(ctx context.Context, buff *rpcmsg.BuffMessage) (*rpcmsg.BackMessage, error) {
	call := &rpcmsg.CallMessage{
		Buff: buff,
	}
	fullName := protoreflect.FullName(buff.Name)
	serve := string(fullName.Parent())
	conn := GetManagerInstance().GetClient(serve)
	logrus.Debugf("Cli => (%v) %s [%v]", call.Mate, call.Buff.Name, call.Buff)
	ret, err := conn.Client.Call(ctx, call)
	if err != nil {
		s := status.Convert(err)
		if s.Code() == codes.DataLoss && s.Message() == "nil" {
			return nil, nil
		}
	}
	logrus.Debugf("Cli <= (%v) %s [%v] %v", call.Mate, call.Buff.Name, ret, err)
	return ret, err
}
