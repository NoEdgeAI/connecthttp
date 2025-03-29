package main

import (
	"flag"
	"fmt"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"
)

const (
	contextPackage = protogen.GoImportPath("context")
	errorsPackage  = protogen.GoImportPath("errors")
	httpPackage    = protogen.GoImportPath("net/http")
	stringsPackage = protogen.GoImportPath("strings")
	connectPackage = protogen.GoImportPath("github.com/scmtble/connecthttp")

	generatedFilenameExtension = ".connecthttp.go"
	defaultPackageSuffix       = "connect"
	packageSuffixFlagName      = "package_suffix"

	usage = "usage"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Fprintln(os.Stdout, "beta")
		os.Exit(0)
	}
	if len(os.Args) == 2 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		fmt.Fprintln(os.Stdout, usage)
		os.Exit(0)
	}
	if len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}
	var flagSet flag.FlagSet
	packageSuffix := flagSet.String(
		packageSuffixFlagName,
		defaultPackageSuffix,
		"Generate files into a sub-package of the package containing the base .pb.go files using the given suffix. An empty suffix denotes to generate into the same package as the base pb.go files.",
	)
	protogen.Options{
		ParamFunc: flagSet.Set,
	}.Run(
		func(plugin *protogen.Plugin) error {
			plugin.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL) | uint64(pluginpb.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS)
			plugin.SupportedEditionsMinimum = descriptorpb.Edition_EDITION_PROTO2
			plugin.SupportedEditionsMaximum = descriptorpb.Edition_EDITION_2023
			for _, file := range plugin.Files {
				if file.Generate {
					generate(plugin, file, *packageSuffix)
				}
			}
			return nil
		},
	)
}

func generate(plugin *protogen.Plugin, file *protogen.File, packageSuffix string) {
	if len(file.Services) == 0 {
		return
	}

	goImportPath := file.GoImportPath
	if packageSuffix != "" {
		if !token.IsIdentifier(packageSuffix) {
			plugin.Error(fmt.Errorf("package_suffix %q is not a valid Go identifier", packageSuffix))
			return
		}
		file.GoPackageName += protogen.GoPackageName(packageSuffix)
		generatedFilenamePrefixToSlash := filepath.ToSlash(file.GeneratedFilenamePrefix)
		file.GeneratedFilenamePrefix = path.Join(
			path.Dir(generatedFilenamePrefixToSlash),
			string(file.GoPackageName),
			path.Base(generatedFilenamePrefixToSlash),
		)
		goImportPath = protogen.GoImportPath(path.Join(
			string(file.GoImportPath),
			string(file.GoPackageName),
		))
	}
	generatedFile := plugin.NewGeneratedFile(
		file.GeneratedFilenamePrefix+generatedFilenameExtension,
		goImportPath,
	)
	if packageSuffix != "" {
		generatedFile.Import(file.GoImportPath)
	}
	generatePreamble(generatedFile, file)
	generateServiceNameConstants(generatedFile, file.Services)
	for _, service := range file.Services {
		generateService(generatedFile, file, service)
	}
}

func generatePreamble(g *protogen.GeneratedFile, file *protogen.File) {
	g.P("package ", file.GoPackageName)
}

func generateServiceNameConstants(g *protogen.GeneratedFile, services []*protogen.Service) {
	var numMethods int
	g.P("const (")
	for _, service := range services {
		constName := fmt.Sprintf("%sName", service.Desc.Name())
		g.P(constName, ` = "`, service.Desc.FullName(), `"`)
		numMethods += len(service.Methods)
	}
	g.P(")")
	g.P()

	if numMethods == 0 {
		return
	}
	g.P("const (")
	for _, service := range services {
		for _, method := range service.Methods {
			g.P(procedureConstName(method), ` = "`, fmt.Sprintf("/%s/%s", service.Desc.FullName(), method.Desc.Name()), `"`)
		}
	}
	g.P(")")
	g.P()
}

func generateService(g *protogen.GeneratedFile, file *protogen.File, service *protogen.Service) {
	names := newNames(service)
	generateServerInterface(g, service, names)
	generateServerConstructor(g, file, service, names)
}

func generateServerInterface(g *protogen.GeneratedFile, service *protogen.Service, names names) {
	if isDeprecatedService(service) {
		g.P("//")
		deprecated(g)
	}
	g.AnnotateSymbol(names.Server, protogen.Annotation{Location: service.Location})
	g.P("type ", names.Server, " interface {")
	for _, method := range service.Methods {
		leadingComments(
			g,
			method.Comments.Leading,
			isDeprecatedMethod(method),
		)
		g.AnnotateSymbol(names.Server+"."+method.GoName, protogen.Annotation{Location: method.Location})
		g.P(serverSignature(g, method))
	}
	g.P("}")
	g.P()
}

func generateServerConstructor(g *protogen.GeneratedFile, _ *protogen.File, service *protogen.Service, names names) {
	g.P("//")
	if isDeprecatedService(service) {
		g.P("//")
		deprecated(g)
	}
	handlerOption := connectPackage.Ident("HandlerOption")
	g.P("func ", names.ServerConstructor, "(svc ", names.Server, ", opts ...", handlerOption,
		") (string, ", httpPackage.Ident("Handler"), ") {")
	for _, method := range service.Methods {
		isStreamingServer := method.Desc.IsStreamingServer()
		isStreamingClient := method.Desc.IsStreamingClient()
		if isStreamingServer || isStreamingClient {
			continue
		}

		g.P(procedureHandlerName(method), ` := `, connectPackage.Ident("NewHandler"), "(")
		g.P(procedureConstName(method), `,`)
		g.P("svc.", method.GoName, ",")
		g.P("opts...,")
		g.P(")")
	}
	g.P(`return "/`, service.Desc.FullName(), `/", `, httpPackage.Ident("HandlerFunc"), `(func(w `, httpPackage.Ident("ResponseWriter"), `, r *`, httpPackage.Ident("Request"), `){`)
	g.P("switch r.URL.Path {")
	for _, method := range service.Methods {
		g.P("case ", procedureConstName(method), ":")
		g.P(procedureHandlerName(method), ".ServeHTTP(w, r)")
	}
	g.P("default:")
	g.P(httpPackage.Ident("NotFound"), "(w, r)")
	g.P("}")
	g.P("})")
	g.P("}")
	g.P()
}

func serverSignature(g *protogen.GeneratedFile, method *protogen.Method) string {
	return method.GoName + serverSignatureParams(g, method, false /* named */)
}

func serverSignatureParams(g *protogen.GeneratedFile, method *protogen.Method, named bool) string {
	ctxName := "ctx "
	reqName := "req "
	streamName := "stream "
	if !named {
		ctxName, reqName, streamName = "", "", ""
	}
	if method.Desc.IsStreamingClient() && method.Desc.IsStreamingServer() {
		// bidi streaming
		return "(" + ctxName + g.QualifiedGoIdent(contextPackage.Ident("Context")) + ", " +
			streamName + "*" + g.QualifiedGoIdent(connectPackage.Ident("BidiStream")) +
			"[" + g.QualifiedGoIdent(method.Input.GoIdent) + ", " + g.QualifiedGoIdent(method.Output.GoIdent) + "]" +
			") error"
	}
	if method.Desc.IsStreamingClient() {
		// client streaming
		return "(" + ctxName + g.QualifiedGoIdent(contextPackage.Ident("Context")) + ", " +
			streamName + "*" + g.QualifiedGoIdent(connectPackage.Ident("ClientStream")) +
			"[" + g.QualifiedGoIdent(method.Input.GoIdent) + "]" +
			") (*" + g.QualifiedGoIdent(connectPackage.Ident("Response")) + "[" + g.QualifiedGoIdent(method.Output.GoIdent) + "] ,error)"
	}
	if method.Desc.IsStreamingServer() {
		// server streaming
		return "(" + ctxName + g.QualifiedGoIdent(contextPackage.Ident("Context")) +
			", " + reqName + "*" + g.QualifiedGoIdent(connectPackage.Ident("Request")) + "[" +
			g.QualifiedGoIdent(method.Input.GoIdent) + "], " +
			streamName + "*" + g.QualifiedGoIdent(connectPackage.Ident("ServerStream")) +
			"[" + g.QualifiedGoIdent(method.Output.GoIdent) + "]" +
			") error"
	}
	// unary
	return "(" + ctxName + g.QualifiedGoIdent(contextPackage.Ident("Context")) +
		", " + reqName + "*" + g.QualifiedGoIdent(connectPackage.Ident("Request")) + "[" +
		g.QualifiedGoIdent(method.Input.GoIdent) + "]) " +
		"(*" + g.QualifiedGoIdent(connectPackage.Ident("Response")) + "[" +
		g.QualifiedGoIdent(method.Output.GoIdent) + "], error)"
}

func procedureConstName(m *protogen.Method) string {
	return fmt.Sprintf("%s%sProcedure", m.Parent.GoName, m.GoName)
}

func procedureHandlerName(m *protogen.Method) string {
	return fmt.Sprintf("%s%sHandler", unexport(m.Parent.GoName), m.GoName)
}

func isDeprecatedService(service *protogen.Service) bool {
	serviceOptions, ok := service.Desc.Options().(*descriptorpb.ServiceOptions)
	return ok && serviceOptions.GetDeprecated()
}

func isDeprecatedMethod(method *protogen.Method) bool {
	methodOptions, ok := method.Desc.Options().(*descriptorpb.MethodOptions)
	return ok && methodOptions.GetDeprecated()
}

func leadingComments(g *protogen.GeneratedFile, comments protogen.Comments, isDeprecated bool) {
	if comments.String() != "" {
		g.P(strings.TrimSpace(comments.String()))
	}
	if isDeprecated {
		if comments.String() != "" {
			g.P("//")
		}
		deprecated(g)
	}
}

func deprecated(g *protogen.GeneratedFile) {
	g.P("// Deprecated: do not use.")
}

func unexport(s string) string {
	lowercased := strings.ToLower(s[:1]) + s[1:]
	switch lowercased {
	// https://go.dev/ref/spec#Keywords
	case "break", "default", "func", "interface", "select",
		"case", "defer", "go", "map", "struct",
		"chan", "else", "goto", "package", "switch",
		"const", "fallthrough", "if", "range", "type",
		"continue", "for", "import", "return", "var":
		return "_" + lowercased
	default:
		return lowercased
	}
}

type names struct {
	Base                string
	Client              string
	ClientConstructor   string
	ClientImpl          string
	ClientExposeMethod  string
	Server              string
	ServerConstructor   string
	UnimplementedServer string
}

func newNames(service *protogen.Service) names {
	base := service.GoName
	return names{
		Base:                base,
		Client:              fmt.Sprintf("%sClient", base),
		ClientConstructor:   fmt.Sprintf("New%sClient", base),
		ClientImpl:          fmt.Sprintf("%sClient", unexport(base)),
		Server:              fmt.Sprintf("%sHandler", base),
		ServerConstructor:   fmt.Sprintf("New%sHandler", base),
		UnimplementedServer: fmt.Sprintf("Unimplemented%sHandler", base),
	}
}
