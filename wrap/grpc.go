package wrap

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/emicklei/proto"
	"gofr.dev/pkg/gofr"
)

const (
	filePerm                = 0644
	serverFileSuffix        = "_server.go"
	serverWrapperFileSuffix = "_gofr.go"
	clientFileSuffix        = "_client.go"
)

var (
	ErrNoProtoFile        = errors.New("proto file path is required")
	ErrOpeningProtoFile   = errors.New("error opening the proto file")
	ErrFailedToParseProto = errors.New("failed to parse proto file")
	ErrGeneratingWrapper  = errors.New("error while generating the code using proto file")
	ErrWritingFile        = errors.New("error writing the generated code to the file")
)

// ServiceMethod represents a method in a proto service.
type ServiceMethod struct {
	Name      string
	Request   string
	Response  string
	Streaming bool
}

// ProtoService represents a service in a proto file.
type ProtoService struct {
	Name    string
	Methods []ServiceMethod
}

// WrapperData is the template data structure.
type WrapperData struct {
	Package  string
	Service  string
	Methods  []ServiceMethod
	Requests []string
}

type FileType struct {
	FileSuffix    string
	CodeGenerator func(*gofr.Context, *WrapperData) string
}

// BuildGRPCGoFrClient generates gRPC client wrapper code based on a proto definition.
func BuildGRPCGoFrClient(ctx *gofr.Context) (any, error) {
	gRPCClient := FileType{
		FileSuffix:    clientFileSuffix,
		CodeGenerator: generateGoFrClient,
	}

	return generateWrapper(ctx, gRPCClient)
}

// BuildGRPCGoFrServer generates gRPC client and server code based on a proto definition.
func BuildGRPCGoFrServer(ctx *gofr.Context) (any, error) {
	gRPCServer := []FileType{
		{FileSuffix: serverWrapperFileSuffix, CodeGenerator: generateGoFrServerWrapper},
		{FileSuffix: serverFileSuffix, CodeGenerator: generateGoFrServer},
	}

	return generateWrapper(ctx, gRPCServer...)
}

// generateWrapper executes the function for specified FileType to create GoFr integrated
// gRPC server/client files with the required services in proto file and
// specified suffix for every service specified in the proto file.
func generateWrapper(ctx *gofr.Context, options ...FileType) (any, error) {
	protoPath := ctx.Param("proto")
	if protoPath == "" {
		return nil, ErrNoProtoFile
	}

	file, err := os.Open(protoPath)
	if err != nil {
		ctx.Errorf("Failed to open proto file: %v", err)
		return nil, ErrOpeningProtoFile
	}
	defer file.Close()

	parser := proto.NewParser(file)

	definition, err := parser.Parse()
	if err != nil {
		ctx.Errorf("Failed to parse proto file: %v", err)
		return nil, ErrFailedToParseProto
	}

	projectPath, packageName := getPackageAndProject(definition, protoPath)
	services := getServices(definition)

	for _, service := range services {
		wrapperData := WrapperData{
			Package:  packageName,
			Service:  service.Name,
			Methods:  service.Methods,
			Requests: uniqueRequestTypes(service.Methods),
		}

		for _, option := range options {
			generatedCode := option.CodeGenerator(ctx, &wrapperData)
			if generatedCode == "" {
				ctx.Errorf("Failed to generate code for service %s with file suffix %s", service.Name, option.FileSuffix)
				return nil, ErrGeneratingWrapper
			}

			// Generate output file path based on service name and file suffix.
			outputFilePath := path.Join(projectPath, strings.ToLower(service.Name)+option.FileSuffix)
			if writeErr := os.WriteFile(outputFilePath, []byte(generatedCode), filePerm); writeErr != nil {
				ctx.Errorf("Failed to write file %s: %v", outputFilePath, writeErr)
				return nil, ErrWritingFile
			}

			fmt.Printf("Generated file for service %s at %s\n", service.Name, outputFilePath)
		}
	}

	return "Successfully generated all files for GoFr integrated gRPC servers/clients", nil
}

// Extract unique request types from methods.
func uniqueRequestTypes(methods []ServiceMethod) []string {
	requests := make(map[string]bool)

	for _, method := range methods {
		if !method.Streaming {
			requests[method.Request] = true
		}
	}

	uniqueRequests := make([]string, 0, len(requests))
	for request := range requests {
		uniqueRequests = append(uniqueRequests, request)
	}

	return uniqueRequests
}

// Generate GoFr server wrapper for gRPC using the wrapperTemplate.
func generateGoFrServerWrapper(ctx *gofr.Context, data *WrapperData) string {
	return executeTemplate(ctx, data, wrapperTemplate)
}

// Generate GoFr gRPCHandler code using the serverTemplate.
func generateGoFrServer(ctx *gofr.Context, data *WrapperData) string {
	return executeTemplate(ctx, data, serverTemplate)
}

// Generate GoFr gRPC Client code using the clientTemplate.
func generateGoFrClient(ctx *gofr.Context, data *WrapperData) string {
	return executeTemplate(ctx, data, clientTemplate)
}

// Execute a template with data.
func executeTemplate(ctx *gofr.Context, data *WrapperData, tmpl string) string {
	var buf bytes.Buffer

	tmplInstance := template.Must(template.New("template").Parse(tmpl))
	if err := tmplInstance.Execute(&buf, data); err != nil {
		ctx.Errorf("Template execution failed: %v", err)
		return ""
	}

	return buf.String()
}

func getPackageAndProject(definition *proto.Proto, protoPath string) (projectPath, packageName string) {
	proto.Walk(definition,
		proto.WithOption(func(opt *proto.Option) {
			if opt.Name == "go_package" {
				packageName = path.Base(opt.Constant.Source)
			}
		}),
	)

	return path.Dir(protoPath), packageName
}

func getServices(definition *proto.Proto) []ProtoService {
	var services []ProtoService

	proto.Walk(definition,
		proto.WithService(func(s *proto.Service) {
			service := ProtoService{Name: s.Name}

			for _, element := range s.Elements {
				if rpc, ok := element.(*proto.RPC); ok {
					method := ServiceMethod{
						Name:      rpc.Name,
						Request:   rpc.RequestType,
						Response:  rpc.ReturnsType,
						Streaming: rpc.StreamsReturns || rpc.StreamsRequest,
					}

					service.Methods = append(service.Methods, method)
				}
			}

			services = append(services, service)
		}),
	)

	return services
}
