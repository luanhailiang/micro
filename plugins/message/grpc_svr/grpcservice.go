package grpc_svr

import (
	"fmt"
	"net"
	"runtime/debug"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/luanhailiang/micro.git/proto/rpcmsg"
)

type Server struct {
	rpcmsg.UnimplementedCommandServer
}

// ServiceCall
func (s *Server) Call(ctx context.Context, cmd *rpcmsg.CallMessage) (*rpcmsg.BackMessage, error) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("grpc Call:%v panic:%s", cmd, err)
			debug.PrintStack()
		}
	}()
	msg, err := Call(ctx, cmd)
	if err != nil {
		log.Errorf("grpc serve err:%s", err.Error())
	} else if msg == nil {
		err = status.Errorf(codes.DataLoss, "nil")
	}
	return msg, err
}

// StartServe 在main函数调用
func StartServe() {
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", 9090))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	rpcmsg.RegisterCommandServer(s, &Server{})
	log.Infof("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
