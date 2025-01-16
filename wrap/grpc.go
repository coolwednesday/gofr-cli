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

const filePerm = 0644

var (
	ErrNoProtoFile              = errors.New("proto file path is required")
	ErrOpeningProtoFile         = errors.New("error opening the proto file")
	ErrFailedToParseProto       = errors.New("failed to parse proto file")
	ErrGeneratingWrapper        = errors.New("error generating the wrapper code from the proto file")
	ErrWritingWrapperFile       = errors.New("error writing the generated wrapper to the file")
	ErrGeneratingServerTemplate = errors.New("error generating the gRPC server file template")
	ErrWritingServerTemplate    = errors.New("error writing the generated server template to the file")
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

// GenerateClientWrapper generates gRPC client wrapper code based on a proto definition.
func GenerateClientWrapper(ctx *gofr.Context) (any, error) {
	return generateWrapperFiles(ctx, "_client.go", generateClientCode)
}

// GenerateWrapper generates gRPC client and server code based on a proto definition.
func GenerateWrapper(ctx *gofr.Context) (any, error) {
	return generateWrapperFiles(ctx, "_gofr.go", generateWrapperCode, "_server.go", generategRPCCode)
}

// Generates wrapper files for specified extensions and generation functions.
func generateWrapperFiles(ctx *gofr.Context, extensionsAndGenerators ...interface{}) (any, error) {
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

		for i := 0; i < len(extensionsAndGenerators); i += 2 {
			extension := extensionsAndGenerators[i].(string)
			generator := extensionsAndGenerators[i+1].(func(*gofr.Context, *WrapperData) string)

			generatedCode := generator(ctx, &wrapperData)
			if generatedCode == "" {
				if extension == "_server.go" {
					ctx.Errorf("%v: %v", ErrGeneratingServerTemplate, service.Name)
					return nil, ErrGeneratingServerTemplate
				}
				return nil, ErrGeneratingWrapper
			}

			outputFilePath := path.Join(projectPath, fmt.Sprintf("%s%s", strings.ToLower(service.Name), extension))
			err := os.WriteFile(outputFilePath, []byte(generatedCode), filePerm)
			if err != nil {
				if extension == "_server.go" {
					ctx.Errorf("%v: %v", ErrWritingServerTemplate, service.Name)
					return nil, ErrWritingServerTemplate
				}
				ctx.Errorf("Failed to write file %s: %v", outputFilePath, err)
				return nil, ErrWritingWrapperFile
			}

			fmt.Printf("Generated file for service %s at %s\n", service.Name, outputFilePath)
		}
	}

	return "Successfully generated all wrappers for gRPC services", nil
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

// Generate wrapper code using the template.
func generateWrapperCode(ctx *gofr.Context, data *WrapperData) string {
	return executeTemplate(ctx, data, wrapperTemplate)
}

// Generate gRPC server code using the template.
func generategRPCCode(ctx *gofr.Context, data *WrapperData) string {
	return executeTemplate(ctx, data, serverTemplate)
}

// Generate client code using the template.
func generateClientCode(ctx *gofr.Context, data *WrapperData) string {
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

func getPackageAndProject(definition *proto.Proto, protoPath string) (string, string) {
	var packageName string
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
