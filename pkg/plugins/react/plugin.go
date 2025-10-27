package react

import (
	"github.com/withgalaxy/galaxy/pkg/plugins"
)

type ReactPlugin struct {
	setupCtx *plugins.SetupContext
}

func New() *ReactPlugin {
	return &ReactPlugin{}
}

func (p *ReactPlugin) Name() string {
	return "react"
}

func (p *ReactPlugin) Setup(ctx *plugins.SetupContext) error {
	p.setupCtx = ctx
	return nil
}

func (p *ReactPlugin) TransformCSS(css string, filePath string) (string, error) {
	return css, nil
}

func (p *ReactPlugin) TransformJS(js string, filePath string) (string, error) {
	return js, nil
}

func (p *ReactPlugin) InjectTags() []plugins.HTMLTag {
	return nil
}

func (p *ReactPlugin) BuildStart(ctx *plugins.BuildContext) error {
	return nil
}

func (p *ReactPlugin) BuildEnd(ctx *plugins.BuildContext) error {
	return nil
}
