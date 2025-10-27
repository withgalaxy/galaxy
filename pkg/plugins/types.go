package plugins

import (
	"github.com/withgalaxy/galaxy/pkg/config"
	"github.com/withgalaxy/galaxy/pkg/parser"
)

type Plugin interface {
	Name() string
	Setup(ctx *SetupContext) error
	TransformCSS(css string, filePath string) (string, error)
	TransformJS(js string, filePath string) (string, error)
	InjectTags() []HTMLTag
	BuildStart(ctx *BuildContext) error
	BuildEnd(ctx *BuildContext) error
}

type SetupContext struct {
	Config    *config.Config
	PluginCfg map[string]interface{}
	RootDir   string
	OutDir    string
}

type BuildContext struct {
	Config     *config.Config
	RootDir    string
	OutDir     string
	PagesDir   string
	PublicDir  string
	Components map[string]*parser.Component
}

type HTMLTag struct {
	Tag        string
	Attributes map[string]string
	Content    string
	Position   TagPosition
}

type TagPosition int

const (
	HeadStart TagPosition = iota
	HeadEnd
	BodyStart
	BodyEnd
)
