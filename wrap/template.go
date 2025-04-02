package wrap

const (
	wrapperTemplate = `// Code generated by gofr.dev/cli/gofr. DO NOT EDIT.
// versions:
// 	gofr-cli v0.6.0
// 	gofr.dev v1.37.0
// 	source: {{ .Source }}

package {{ .Package }}

import (
	"context"

	"gofr.dev/pkg/gofr"
	"gofr.dev/pkg/gofr/container"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// New{{ .Service }}GoFrServer creates a new instance of {{ .Service }}GoFrServer
func New{{ .Service }}GoFrServer() *{{ .Service }}GoFrServer {
	return &{{ .Service }}GoFrServer{
		health: getOrCreateHealthServer(), // Initialize the health server
	}
}

// {{ .Service }}ServerWithGofr is the interface for the server implementation
type {{ .Service }}ServerWithGofr interface {
	{{- range .Methods }}
	{{ .Name }}(*gofr.Context) (any, error)
	{{- end }}
}

// {{ .Service }}ServerWrapper wraps the server and handles request and response logic
type {{ .Service }}ServerWrapper struct {
	{{ .Service }}Server
	*healthServer
	Container *container.Container
	server    {{ .Service }}ServerWithGofr
}

// {{- range .Methods }}
{{- if not .Streaming }}
// {{ .Name }} wraps the method and handles its execution
func (h *{{ $.Service }}ServerWrapper) {{ .Name }}(ctx context.Context, req *{{ .Request }}) (*{{ .Response }}, error) {
	gctx := h.getGofrContext(ctx, &{{ .Request }}Wrapper{ctx: ctx, {{ .Request }}: req})

	res, err := h.server.{{ .Name }}(gctx)
	if err != nil {
		return nil, err
	}

	resp, ok := res.(*{{ .Response }})
	if !ok {
		return nil, status.Errorf(codes.Unknown, "unexpected response type %T", res)
	}

	return resp, nil
}
{{- end }}
{{- end }}

// mustEmbedUnimplemented{{ .Service }}Server ensures that the server implements all required methods
func (h *{{ .Service }}ServerWrapper) mustEmbedUnimplemented{{ .Service }}Server() {}

// Register{{ .Service }}ServerWithGofr registers the server with the application
func Register{{ .Service }}ServerWithGofr(app *gofr.App, srv {{ .Service }}ServerWithGofr) {
	registerServerWithGofr(app, srv, func(s grpc.ServiceRegistrar, srv any) {
		wrapper := &{{ .Service }}ServerWrapper{server: srv.({{ .Service }}ServerWithGofr), healthServer: getOrCreateHealthServer()}
		Register{{ .Service }}Server(s, wrapper)
		wrapper.Server.SetServingStatus("{{ .Service }}", healthpb.HealthCheckResponse_SERVING)
	})
}

// getGofrContext extracts the GoFr context from the original context
func (h *{{ .Service }}ServerWrapper) getGofrContext(ctx context.Context, req gofr.Request) *gofr.Context {
	return &gofr.Context{
		Context:   ctx,
		Container: h.Container,
		Request:   req,
	}
}
`

	messageTemplate = `// Code generated by gofr.dev/cli/gofr. DO NOT EDIT.
// versions:
// 	gofr-cli v0.6.0
// 	gofr.dev v1.37.0
// 	source: {{ .Source }}


package {{ .Package }}

import (
	"context"
	"fmt"
	"reflect"
)

// Request Wrappers
{{- range $request := .Requests }}
type {{ $request }}Wrapper struct {
	ctx context.Context
	*{{ $request }}
}

func (h *{{ $request }}Wrapper) Context() context.Context {
	return h.ctx
}

func (h *{{ $request }}Wrapper) Param(s string) string {
	return ""
}

func (h *{{ $request }}Wrapper) PathParam(s string) string {
	return ""
}

func (h *{{ $request }}Wrapper) Bind(p interface{}) error {
	ptr := reflect.ValueOf(p)
	if ptr.Kind() != reflect.Ptr {
		return fmt.Errorf("expected a pointer, got %T", p)
	}

	hValue := reflect.ValueOf(h.{{ $request }}).Elem()
	ptrValue := ptr.Elem()

	for i := 0; i < hValue.NumField(); i++ {
		field := hValue.Type().Field(i)
		if field.Name == "state" || field.Name == "sizeCache" || field.Name == "unknownFields" {
			continue
		}

		if field.IsExported() {
			ptrValue.Field(i).Set(hValue.Field(i))
		}
	}

	return nil
}

func (h *{{ $request }}Wrapper) HostName() string {
	return ""
}

func (h *{{ $request }}Wrapper) Params(s string) []string {
	return nil
}
{{- end }}`

	serverTemplate = `package {{ .Package }}
// versions:
// 	gofr-cli v0.6.0
// 	gofr.dev v1.37.0
// 	source: {{ .Source }}

import "gofr.dev/pkg/gofr"

// Register the gRPC service in your app using the following code in your main.go:
//
// {{ .Package }}.Register{{ $.Service }}ServerWithGofr(app, &{{ .Package }}.New{{ $.Service }}GoFrServer())
//
// {{ $.Service }}GoFrServer defines the gRPC server implementation.
// Customize the struct with required dependencies and fields as needed.

type {{ $.Service }}GoFrServer struct {
 health *healthServer
}

{{- range .Methods }}
func (s *{{ $.Service }}GoFrServer) {{ .Name }}(ctx *gofr.Context) (any, error) {
// Uncomment and use the following code if you need to bind the request payload
// request := {{ .Request }}{}
// err := ctx.Bind(&request)
// if err != nil {
//     return nil, err
// }

return &{{ .Response }}{}, nil
}
{{- end }}
`
	clientTemplate = `// Code generated by gofr.dev/cli/gofr. DO NOT EDIT.
// versions:
// 	gofr-cli v0.6.0
// 	gofr.dev v1.37.0
// 	source: {{ .Source }}

package {{ .Package }}

import (
	"gofr.dev/pkg/gofr"
	"gofr.dev/pkg/gofr/metrics"
	"google.golang.org/grpc"
)

type {{ .Service }}GoFrClient interface {
{{- range .Methods }}
	{{ .Name }}(*gofr.Context, *{{ .Request }}, ...grpc.CallOption) (*{{ .Response }}, error)
{{- end }}
	HealthClient
}

type {{ .Service }}ClientWrapper struct {
	client {{ .Service }}Client
	HealthClient
}

func New{{ .Service }}GoFrClient(host string, metrics metrics.Manager, dialOptions ...grpc.DialOption) ({{ .Service }}GoFrClient, error) {
	conn, err := createGRPCConn(host, "{{ .Service }}", dialOptions...)
	if err != nil {
		return &{{ .Service }}ClientWrapper{
			client:       nil,
			HealthClient: &HealthClientWrapper{client: nil}, // Ensure HealthClient is also implemented
		}, err
	}

	metricsOnce.Do(func() {
		metrics.NewHistogram("app_gRPC-Client_stats", "Response time of gRPC client in milliseconds.", gRPCBuckets...)
	})

	res := New{{ .Service }}Client(conn)
	healthClient := NewHealthClient(conn)

	return &{{ .Service }}ClientWrapper{
		client: res,
		HealthClient: healthClient,
	}, nil
}

{{- range .Methods }}
func (h *{{ $.Service }}ClientWrapper) {{ .Name }}(ctx *gofr.Context, req *{{ .Request }}, 
opts ...grpc.CallOption) (*{{ .Response }}, error) {
	result, err := invokeRPC(ctx, "/{{ $.Service }}/{{ .Name }}", func() (interface{}, error) {
		return h.client.{{ .Name }}(ctx.Context, req, opts...)
	})

	if err != nil {
		return nil, err
	}
	return result.(*{{ .Response }}), nil
}
{{- end }}
`

	healthServerTemplate = `// Code generated by gofr.dev/cli/gofr. DO NOT EDIT.
// versions:
// 	gofr-cli v0.6.0
// 	gofr.dev v1.37.0
// 	source: {{ .Source }}

package {{ .Package }}

import (
	"fmt"
	"google.golang.org/grpc"
	"time"

	"gofr.dev/pkg/gofr"

	gofrGRPC "gofr.dev/pkg/gofr/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type healthServer struct {
	*health.Server
}

var globalHealthServer *healthServer
var healthServerRegistered bool // Global flag to track if health server is registered

// getOrCreateHealthServer ensures only one health server is created and reused.
func getOrCreateHealthServer() *healthServer {
	if globalHealthServer == nil {
		globalHealthServer = &healthServer{health.NewServer()}
	}
	return globalHealthServer
}

func registerServerWithGofr(app *gofr.App, srv any, registerFunc func(grpc.ServiceRegistrar, any)) {
	var s grpc.ServiceRegistrar = app
	h := getOrCreateHealthServer()

	// Register metrics and health server only once
	if !healthServerRegistered {
		gRPCBuckets := []float64{0.005, 0.01, .05, .075, .1, .125, .15, .2, .3, .5, .75, 1, 2, 3, 4, 5, 7.5, 10}
		app.Metrics().NewHistogram("app_gRPC-Server_stats", "Response time of gRPC server in milliseconds.", gRPCBuckets...)

		healthpb.RegisterHealthServer(s, h.Server)
		h.Server.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
		healthServerRegistered = true
	}

	// Register the provided server
	registerFunc(s, srv)
}

func (h *healthServer) Check(ctx *gofr.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	start := time.Now()
	span := ctx.Trace("/grpc.health.v1.Health/Check")
	res, err := h.Server.Check(ctx.Context, req)
	logger := gofrGRPC.NewgRPCLogger()
	logger.DocumentRPCLog(ctx.Context, ctx.Logger, ctx.Metrics(), start, err,
	fmt.Sprintf("/grpc.health.v1.Health/Check	Service: %q", req.Service), "app_gRPC-Server_stats")
	span.End()
	return res, err
}

func (h *healthServer) Watch(ctx *gofr.Context, in *healthpb.HealthCheckRequest, stream healthpb.Health_WatchServer) error {
	start := time.Now()
	span := ctx.Trace("/grpc.health.v1.Health/Watch")
	err := h.Server.Watch(in, stream)
	logger := gofrGRPC.NewgRPCLogger()
	logger.DocumentRPCLog(ctx.Context, ctx.Logger, ctx.Metrics(), start, err,
	fmt.Sprintf("/grpc.health.v1.Health/Watch	Service: %q", in.Service), "app_gRPC-Server_stats")
	span.End()
	return err
}

func (h *healthServer) SetServingStatus(ctx *gofr.Context, service string, servingStatus healthpb.HealthCheckResponse_ServingStatus) {
	start := time.Now()
	span := ctx.Trace("/grpc.health.v1.Health/SetServingStatus")
	h.Server.SetServingStatus(service, servingStatus)
	logger := gofrGRPC.NewgRPCLogger()
	logger.DocumentRPCLog(ctx.Context, ctx.Logger, ctx.Metrics(), start, nil,
	fmt.Sprintf("/grpc.health.v1.Health/SetServingStatus	Service: %q", service), "app_gRPC-Server_stats")
	span.End()
}

func (h *healthServer) Shutdown(ctx *gofr.Context) {
	start := time.Now()
	span := ctx.Trace("/grpc.health.v1.Health/Shutdown")
	h.Server.Shutdown()
	logger := gofrGRPC.NewgRPCLogger()
	logger.DocumentRPCLog(ctx.Context, ctx.Logger, ctx.Metrics(), start, nil,
	"/grpc.health.v1.Health/Shutdown", "app_gRPC-Server_stats")
	span.End()
}

func (h *healthServer) Resume(ctx *gofr.Context) {
	start := time.Now()
	span := ctx.Trace("/grpc.health.v1.Health/Resume")
	h.Server.Resume()
	logger := gofrGRPC.NewgRPCLogger()
	logger.DocumentRPCLog(ctx.Context, ctx.Logger, ctx.Metrics(), start, nil,
	"/grpc.health.v1.Health/Resume", "app_gRPC-Server_stats")
	span.End()
}
`

	clientHealthTemplate = `// Code generated by gofr.dev/cli/gofr. DO NOT EDIT.
// versions:
// 	gofr-cli v0.6.0
// 	gofr.dev v1.37.0
// 	source: {{ .Source }}

package {{ .Package }}

import (
	"fmt"
	"sync"
	"time"

	"gofr.dev/pkg/gofr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"

	gofrgRPC "gofr.dev/pkg/gofr/grpc"
)

var (
	metricsOnce sync.Once
	gRPCBuckets = []float64{0.005, 0.01, .05, .075, .1, .125, .15, .2, .3, .5, .75, 1, 2, 3, 4, 5, 7.5, 10}
)

type HealthClient interface {
	Check(ctx *gofr.Context, in *grpc_health_v1.HealthCheckRequest, opts ...grpc.CallOption) (*grpc_health_v1.HealthCheckResponse, error)
	Watch(ctx *gofr.Context, in *grpc_health_v1.HealthCheckRequest, opts ...grpc.CallOption) (
	grpc.ServerStreamingClient[grpc_health_v1.HealthCheckResponse], error)
}

type HealthClientWrapper struct {
	client grpc_health_v1.HealthClient
}

func NewHealthClient(conn *grpc.ClientConn) HealthClient {
	return &HealthClientWrapper{
		client: grpc_health_v1.NewHealthClient(conn),
	}
}

func createGRPCConn(host string, serviceName string, dialOptions ...grpc.DialOption) (*grpc.ClientConn, error) {
	serviceConfig := ` + "`{\"loadBalancingPolicy\": \"round_robin\"}`" + `

	defaultOpts := []grpc.DialOption{
		grpc.WithDefaultServiceConfig(serviceConfig),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	// Developer Note: If the user provides custom DialOptions, they will override the default options due to 
	// the ordering of dialOptions. This behavior is intentional to ensure the gRPC client connection is properly 
	// configured even when the user does not specify any DialOptions.
	dialOptions = append(defaultOpts, dialOptions...)

	conn, err := grpc.NewClient(host, dialOptions...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func invokeRPC(ctx *gofr.Context, rpcName string, rpcFunc func() (interface{}, error)) (interface{}, error) {
	span := ctx.Trace("gRPC-srv-call: " + rpcName)
	defer span.End()

	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()
	md := metadata.Pairs("x-gofr-traceid", traceID, "x-gofr-spanid", spanID)

	ctx.Context = metadata.NewOutgoingContext(ctx.Context, md)
	transactionStartTime := time.Now()

	res, err := rpcFunc()
	logger := gofrgRPC.NewgRPCLogger()
	logger.DocumentRPCLog(ctx.Context, ctx.Logger, ctx.Metrics(), transactionStartTime, err,
	rpcName, "app_gRPC-Client_stats")

	return res, err
}

func (h *HealthClientWrapper) Check(ctx *gofr.Context, in *grpc_health_v1.HealthCheckRequest, 
	opts ...grpc.CallOption) (*grpc_health_v1.HealthCheckResponse, error) {
	result, err := invokeRPC(ctx, fmt.Sprintf("/grpc.health.v1.Health/Check	Service: %q", in.Service), func() (interface{}, error) {
		return h.client.Check(ctx, in, opts...)
	})

	if err != nil {
		return nil, err
	}
	return result.(*grpc_health_v1.HealthCheckResponse), nil
}

func (h *HealthClientWrapper) Watch(ctx *gofr.Context, in *grpc_health_v1.HealthCheckRequest, 
	opts ...grpc.CallOption) (grpc.ServerStreamingClient[grpc_health_v1.HealthCheckResponse], error) {
	result, err := invokeRPC(ctx, fmt.Sprintf("/grpc.health.v1.Health/Watch	Service: %q", in.Service), func() (interface{}, error) {
		return h.client.Watch(ctx, in, opts...)
	})

	if err != nil {
		return nil, err
	}

	return result.(grpc.ServerStreamingClient[grpc_health_v1.HealthCheckResponse]), nil
}
`
)
