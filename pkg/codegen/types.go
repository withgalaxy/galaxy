package codegen

import (
	"github.com/cameron-webmatter/galaxy/pkg/parser"
	"github.com/cameron-webmatter/galaxy/pkg/router"
)

type HandlerGenerator struct {
	Component  *parser.Component
	Route      *router.Route
	ModuleName string
	BaseDir    string
	CSSPath    string
}

type GeneratedHandler struct {
	PackageName  string
	Imports      []string
	FunctionName string
	Code         string
}

type EndpointHandler struct {
	Route       *router.Route
	Methods     []string
	PackageName string
	ImportPath  string
}

type MainGenerator struct {
	Handlers      []*GeneratedHandler
	Endpoints     []*EndpointHandler
	Routes        []*router.Route
	ModuleName    string
	ManifestPath  string
	HasMiddleware bool
}
