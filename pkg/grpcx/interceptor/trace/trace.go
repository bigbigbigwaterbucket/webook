package trace

import (
	"context"
	"learning_go/webook/pkg/grpcx/interceptor"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type TraceInterceptorBuilder struct {
	interceptor.Builder
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
}

func (t *TraceInterceptorBuilder) BuildServer() grpc.UnaryServerInterceptor {
	if t.tracer == nil {
		t.tracer = otel.Tracer("learning_go/webook/pkg/grpcx/interceptor/trace")
	}
	if t.propagator == nil {
		t.propagator = otel.GetTextMapPropagator()
	}
	attrs := []attribute.KeyValue{
		attribute.Key("rpc.grpc.kind").String("unary"),
		attribute.Key("rpc.component").String("server"),
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		ctx = Extract(ctx, t.propagator)
		ctx, span := t.tracer.Start(ctx, info.FullMethod, trace.WithAttributes(attrs...), trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()
		//semconv.RPCMethodKey.String(info.FullMethod),
		//semconv.NetPeerNameKey.String(t.PeerName(ctx)),
		span.SetAttributes(
			attribute.Key("rpc.method").String(info.FullMethod),
			attribute.Key("net.peer.name").String(t.PeerName(ctx)),
			attribute.Key("net.peer.ip").String(t.PeerIP(ctx)))
		defer func() {
			if err != nil {
				span.RecordError(err)
			} else {
				span.SetStatus(codes.Ok, "OK")
			}
		}()
		resp, err = handler(ctx, req)
		return
	}
}

func (t *TraceInterceptorBuilder) BuildClient() grpc.UnaryClientInterceptor {
	if t.tracer == nil {
		t.tracer = otel.Tracer("learning_go/webook/pkg/grpcx/interceptor/trace")
	}
	if t.propagator == nil {
		t.propagator = otel.GetTextMapPropagator()
	}
	attrs := []attribute.KeyValue{
		attribute.Key("rpc.grpc.kind").String("unary"),
		attribute.Key("rpc.component").String("client"),
	}
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {
		ctx, span := t.tracer.Start(ctx, method, trace.WithAttributes(attrs...), trace.WithSpanKind(trace.SpanKindClient))
		defer span.End()
		//semconv.RPCMethodKey.String(info.FullMethod),
		//semconv.NetPeerNameKey.String(t.PeerName(ctx)),
		span.SetAttributes(
			attribute.Key("rpc.method").String(method),
			attribute.Key("net.peer.name").String(t.PeerName(ctx)),
			attribute.Key("net.peer.ip").String(t.PeerIP(ctx)))
		defer func() {
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			} else {
				span.SetStatus(codes.Ok, "OK")
			}
		}()
		ctx = Inject(ctx, t.propagator)
		err = invoker(ctx, method, req, reply, cc, opts...)
		return
	}
}

func Inject(ctx context.Context, p propagation.TextMapPropagator) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(map[string]string{})
	}
	//这里并没有改变ctx，而只是从ctx中拿到相关数据以此来写入metadata里
	p.Inject(ctx, GrpcHeaderCarrier(md))
	//因此还要再初始化ctx
	return metadata.NewOutgoingContext(ctx, md)
}

func Extract(ctx context.Context, p propagation.TextMapPropagator) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.New(map[string]string{})
	}
	return p.Extract(ctx, GrpcHeaderCarrier(md))
}

type GrpcHeaderCarrier metadata.MD

// Get returns the value associated with the passed key.
func (mc GrpcHeaderCarrier) Get(key string) string {
	vals := metadata.MD(mc).Get(key)
	if len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// Set stores the key-value pair.
func (mc GrpcHeaderCarrier) Set(key string, value string) {
	metadata.MD(mc).Set(key, value)
}

// Keys lists the keys stored in this carrier.
func (mc GrpcHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(mc))
	for k := range metadata.MD(mc) {
		keys = append(keys, k)
	}
	return keys
}
