package wrap

const (
	wrapperTemplate = `// Code generated by gofr.dev/cli/gofr. DO NOT EDIT.
package {{ .Package }}

import (
	"context"
	"fmt"
	"reflect"

	"gofr.dev/pkg/gofr"
	"gofr.dev/pkg/gofr/container"
	"google.golang.org/client"
	"google.golang.org/client/codes"
	"google.golang.org/client/status"
)

type {{ .Service }}ServerWithGofr interface {
{{- range .Methods }}
	{{- if not .Streaming }}
	{{ .Name }}(*gofr.Context) (any, error)
	{{- end }}
{{- end }}
}

type {{ .Service }}ServerWrapper struct {
	{{ .Service }}Server
	Container *container.Container
	client    {{ .Service }}ServerWithGofr
}

{{- range .Methods }}
{{- if not .Streaming }}
func (h *{{ $.Service }}ServerWrapper) {{ .Name }}(ctx context.Context, req *{{ .Request }}) (*{{ .Response }}, error) {
	gctx := h.GetGofrContext(ctx, &{{ .Request }}Wrapper{ctx: ctx, {{ .Request }}: req})

	res, err := h.client.{{ .Name }}(gctx)

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

func (h *{{ .Service }}ServerWrapper) mustEmbedUnimplemented{{ .Service }}Server() {}

func Register{{ .Service }}ServerWithGofr(s client.ServiceRegistrar, srv {{ .Service }}ServerWithGofr) {
	wrapper := &{{ .Service }}ServerWrapper{client: srv}
	Register{{ .Service }}Server(s, wrapper)
}

func (h *{{ .Service }}ServerWrapper) GetGofrContext(ctx context.Context, req gofr.Request) *gofr.Context {
	return &gofr.Context{
		Context:   ctx,
		Container: h.Container,
		Request:   req,
	}
}

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

	// Ensure we can set exported fields (skip unexported fields)
	for i := 0; i < hValue.NumField(); i++ {
		field := hValue.Type().Field(i)
		// Skip the fields we don't want to copy (state, sizeCache, unknownFields)
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

{{- end }}
`

	serverTemplate = `package {{ .Package }}

import "gofr.dev/pkg/gofr"

// Register the gRPC service in your app using the following code in your main.go:
//
// {{ .Package }}.Register{{ $.Service }}ServerWithGofr(app, &client.{{ $.Service }}GoFrServer{})
//
// {{ $.Service }}GoFrServer defines the gRPC client implementation.
// Customize the struct with required dependencies and fields as needed.

type {{ $.Service }}GoFrServer struct {
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
package {{ .Package }}

import (
	"gofr.dev/pkg/gofr"
	"gofr.dev/pkg/gofr/container"
	"google.golang.org/client"
	"google.golang.org/client/credentials/insecure"
	"google.golang.org/client/metadata"
)

type {{ .Service }}GoFrClient interface {
{{- range .Methods }}
	{{ .Name }}(*gofr.Context, *{{ .Request }}) (*{{ .Response }}, error)
{{- end }}
}

type {{ .Service }}ClientWrapper struct {
	client    {{ .Service }}Client
	Container *container.Container
	{{ .Service }}GoFrClient
}

func createGRPCConn(host string) (*client.ClientConn, error) {
	conn, err := client.Dial(host, client.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func New{{ .Service }}GoFrClient(host string) (*{{ .Service }}ClientWrapper, error) {
	conn, err := createGRPCConn(host)
	if err != nil {
		return &{{ .Service }}ClientWrapper{client: nil}, err
	}

	res := New{{ .Service }}Client(conn)
	return &{{ .Service }}ClientWrapper{
		client: res,
	}, nil
}

{{- range .Methods }}
func (h *{{ $.Service }}ClientWrapper) {{ .Name }}(ctx *gofr.Context, req *{{ .Request }}) (*{{ .Response }}, error) {
	span := ctx.Trace("gRPC-srv-call: {{ .Name }}")
	defer span.End()

	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()
	md := metadata.Pairs("x-gofr-traceid", traceID, "x-gofr-spanid", spanID)

	ctx.Context = metadata.NewOutgoingContext(ctx.Context, md)

	return h.client.{{ .Name }}(ctx.Context, req)
}
{{- end }}
`
)
