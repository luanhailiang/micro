package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"

	"github.com/luanhailiang/micro/plugins/matedata"
	"github.com/luanhailiang/micro/plugins/message/grpc_cli"
	"github.com/luanhailiang/micro/proto/rpcmsg"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func tracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		tracingHeaders := []string{
			// All applications should propagate x-request-id. This header is
			// included in access log statements and is used for consistent trace
			// sampling and log sampling decisions in Istio.
			"x-request-id",

			// Lightstep tracing header. Propagate this if you use lightstep tracing
			// in Istio (see
			// https://istio.io/latest/docs/tasks/observability/distributed-tracing/lightstep/)
			// Note: this should probably be changed to use B3 or W3C TRACE_CONTEXT.
			// Lightstep recommends using B3 or TRACE_CONTEXT and most application
			// libraries from lightstep do not support x-ot-span-context.
			"x-ot-span-context",

			// Datadog tracing header. Propagate these headers if you use Datadog
			// tracing.
			"x-datadog-trace-id",
			"x-datadog-parent-id",
			"x-datadog-sampling-priority",

			// W3C Trace Context. Compatible with OpenCensusAgent and Stackdriver Istio
			// configurations.
			"traceparent",
			"tracestate",

			// Cloud trace context. Compatible with OpenCensusAgent and Stackdriver Istio
			// configurations.
			"x-cloud-trace-context",

			// Grpc binary trace context. Compatible with OpenCensusAgent nad
			// Stackdriver Istio configurations.
			"grpc-trace-bin",

			// b3 trace headers. Compatible with Zipkin, OpenCensusAgent, and
			// Stackdriver Istio configurations. Commented out since they are
			// propagated by the OpenTracing tracer above.
			"x-b3-traceid",
			"x-b3-spanid",
			"x-b3-parentspanid",
			"x-b3-sampled",
			"x-b3-flags",

			// Application-specific headers to forward.
			"end-user",
			"user-agent",

			// Context and session specific headers
			"cookie",
			"authorization",
			"jwt",
			"meta",
		}
		for _, key := range tracingHeaders {
			if val := c.GetHeader(key); val != "" {
				ctx = metadata.AppendToOutgoingContext(ctx, key, val)
				logrus.Debug("tracingMiddleware", key, val)
			}
		}
		c.Request.WithContext(ctx)
		c.Next()
	}
}

// StartServe 在main函数调用
func StartServe() {
	router := gin.Default()

	router.Use(CORSMiddleware())
	router.Use(tracingMiddleware())
	//对象调用
	router.POST("/proto/:fullname", func(c *gin.Context) {
		isJson := c.GetHeader("Content-Type") != "application/x-protobuf"

		fullname := c.Param("fullname")
		body, err := c.GetRawData()
		if err != nil {
			if isJson {
				c.JSON(http.StatusOK, gin.H{"code": 1, "info": err.Error()})
			} else {
				c.ProtoBuf(http.StatusOK, &rpcmsg.BackMessage{Code: 1, Info: err.Error()})
			}
			return
		}

		//要从jwt里获取防止修改
		var mate *rpcmsg.MateMessage
		token := c.GetHeader("Authorization")
		if token != "" {
			token = strings.Replace(token, "Bearer ", "", 1)
			mate, err = matedata.JwtMate(token)
			if err != nil {
				if isJson {
					c.JSON(http.StatusOK, gin.H{"code": 1, "info": err.Error()})
				} else {
					c.ProtoBuf(http.StatusOK, &rpcmsg.BackMessage{Code: 1, Info: err.Error()})
				}
				logrus.Debugf("error:%s token:%s", err.Error(), token)
				return
			}
		} else if os.Getenv("IS_DEBUG") != "" {
			mate = &rpcmsg.MateMessage{
				Space: c.GetHeader("space"),
				Index: c.GetHeader("index"),
				Usrid: c.GetHeader("usrid"),
				Admin: c.GetHeader("admin"),
			}
		} else if os.Getenv("IS_MATCH") != "" {
			matched, _ := regexp.MatchString(os.Getenv("IS_MATCH"), fullname)
			if matched {
				mate = &rpcmsg.MateMessage{}
			}
		}

		if mate == nil {
			if isJson {
				c.JSON(http.StatusOK, gin.H{"code": 1, "info": "token miss"})
			} else {
				c.ProtoBuf(http.StatusOK, &rpcmsg.BackMessage{Code: 1, Info: "token miss"})
			}
			return
		}

		buff := &rpcmsg.BuffMessage{
			Name: fullname,
			Data: body,
			Json: isJson,
		}
		logrus.Debugf("cmd => %v %v", mate, buff)
		ctx := matedata.NewMateContext(c.Request.Context(), mate)
		back, err := grpc_cli.CallBuff(ctx, buff)
		logrus.Debugf("cmd <= %v %v", mate, back)
		if err != nil {
			if isJson {
				c.JSON(http.StatusOK, gin.H{"code": 1, "info": err.Error()})
			} else {
				c.ProtoBuf(http.StatusOK, &rpcmsg.BackMessage{Code: 1, Info: err.Error()})
			}
			return
		}
		if back == nil {
			back = &rpcmsg.BackMessage{}
		}

		if !isJson {
			c.ProtoBuf(http.StatusOK, back)
			return
		}

		if back.GetBuff() == nil {
			c.JSON(http.StatusOK, gin.H{
				"code": back.Code,
				"info": back.Info,
			})
			return
		}
		data := &gin.H{}
		err = json.Unmarshal(back.GetBuff().Data, data)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 1, "info": err.Error()})
			return
		}

		// c.Request.Response.Header.Add("name", back.GetBuff().Name)
		c.JSON(http.StatusOK, gin.H{
			"code": back.Code,
			"info": back.Info,
			"data": data,
		})
	})
	//启动服务
	router.Run(fmt.Sprintf(":%d", 8080))
}
