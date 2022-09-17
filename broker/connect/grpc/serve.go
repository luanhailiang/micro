package main

import (
	"context"
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/luanhailiang/micro/plugins/matedata"
	"github.com/luanhailiang/micro/plugins/message/grpc_cli"
	"github.com/luanhailiang/micro/proto/rpcmsg"
	"github.com/sirupsen/logrus"
)

type Server struct {
	rpcmsg.UnimplementedConnectServer
}

// ServiceCall
func (s *Server) Link(stream rpcmsg.Connect_LinkServer) error {
	c := NewConnecter(stream)
	return c.IOLoop()
}

// ServiceCall
func (s *Server) Exec(ctx context.Context, buff *rpcmsg.BuffMessage) (*rpcmsg.BackMessage, error) {
	//要从jwt里获取防止修改
	mate := &rpcmsg.MateMessage{}
	logrus.Debugf("cmd => %v %v", mate, buff)
	ctx = matedata.NewMateContext(ctx, mate)
	back, err := grpc_cli.CallBuff(ctx, buff)
	logrus.Debugf("cmd <= %v %v", mate, back)
	return back, err
}

// StartServe 在main函数调用
func StartServe() {
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", 9090))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	rpcmsg.RegisterConnectServer(s, &Server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
