package svelte

import (
	"github.com/withgalaxy/galaxy/pkg/plugins"
)

type SveltePlugin struct {
	setupCtx *plugins.SetupContext
}

func New() *SveltePlugin {
	return &SveltePlugin{}
}

func (p *SveltePlugin) Name() string {
	return "svelte"
}

func (p *SveltePlugin) Setup(ctx *plugins.SetupContext) error {
	p.setupCtx = ctx
	return nil
}

func (p *SveltePlugin) TransformCSS(css string, filePath string) (string, error) {
	return css, nil
}

func (p *SveltePlugin) TransformJS(js string, filePath string) (string, error) {
	return js, nil
}

func (p *SveltePlugin) InjectTags() []plugins.HTMLTag {
	return nil
}

func (p *SveltePlugin) BuildStart(ctx *plugins.BuildContext) error {
	return nil
}

func (p *SveltePlugin) BuildEnd(ctx *plugins.BuildContext) error {
	return nil
}
