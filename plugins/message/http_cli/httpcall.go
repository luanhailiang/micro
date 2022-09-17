package http_cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/luanhailiang/micro.git/proto/rpcmsg"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

func TransformHeader(ctx context.Context, reqest *http.Request) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return
	}
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
		if val := md.Get(key); len(val) != 0 {
			reqest.Header.Add(key, val[0])
			logrus.Debug("tracingMiddleware", key, val)
		}
	}
}

func HttpGet(ctx context.Context, url string) ([]byte, error) {
	client := &http.Client{}
	reqest, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Errorf("error httpget :%s", err.Error())
		return nil, err
	}
	TransformHeader(ctx, reqest)
	resp, err := client.Do(reqest)
	if err != nil {
		log.Errorf("error httpget :%s", err.Error())
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("error httpget read :%s", err.Error())
		return nil, err
	}

	if l := len(body); l > 50 {
		log.Infof("httpget reqs: %s resp : len: %d", url, l)
	} else {
		log.Infof("httpget reqs: %s resp : %s", url, string(body))
	}

	return body, nil
}

func HttpPost(ctx context.Context, url string, obj interface{}) ([]byte, error) {
	var err error
	var data []byte
	switch obj.(type) {
	case string:
		data = []byte(obj.(string))
	case []byte:
		data = obj.([]byte)
	default:
		data, err = json.Marshal(obj)
	}
	if err != nil {
		log.Errorf("error httppost :%s", err.Error())
		return nil, err
	}

	client := &http.Client{}
	reqest, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		log.Errorf("error httpget :%s", err.Error())
		return nil, err
	}
	TransformHeader(ctx, reqest)
	reqest.Header.Set("Content-Type", "application/json;charset=UTF-8")
	resp, err := client.Do(reqest)

	if err != nil {
		log.Errorf("error httppost :%s", err.Error())
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("error httppost read :%s", err.Error())
		return nil, err
	}
	return body, nil
}

func HttpPostProto(ctx context.Context, url string, req proto.Message) (*rpcmsg.BackMessage, error) {
	url += string(req.ProtoReflect().Descriptor().FullName())
	data, err := proto.Marshal(req)
	if err != nil {
		log.Errorf("error HttpPostProto :%s", err.Error())
		return nil, err
	}
	client := &http.Client{}
	reqest, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		log.Errorf("error httpget :%s", err.Error())
		return nil, err
	}
	TransformHeader(ctx, reqest)
	reqest.Header.Set("Content-Type", "application/x-protobuf")
	resp, err := client.Do(reqest)

	if err != nil {
		log.Errorf("error HttpPostProto :%s", err.Error())
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("error HttpPostProto :%s", err.Error())
		return nil, err
	}
	back := &rpcmsg.BackMessage{}
	proto.Unmarshal(body, back)
	return back, nil
}
