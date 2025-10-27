package vue

import (
	"github.com/withgalaxy/galaxy/pkg/plugins"
)

type VuePlugin struct {
	setupCtx *plugins.SetupContext
}

func New() *VuePlugin {
	return &VuePlugin{}
}

func (p *VuePlugin) Name() string {
	return "vue"
}

func (p *VuePlugin) Setup(ctx *plugins.SetupContext) error {
	p.setupCtx = ctx
	return nil
}

func (p *VuePlugin) TransformCSS(css string, filePath string) (string, error) {
	return css, nil
}

func (p *VuePlugin) TransformJS(js string, filePath string) (string, error) {
	return js, nil
}

func (p *VuePlugin) InjectTags() []plugins.HTMLTag {
	return nil
}

func (p *VuePlugin) BuildStart(ctx *plugins.BuildContext) error {
	return nil
}

func (p *VuePlugin) BuildEnd(ctx *plugins.BuildContext) error {
	return nil
}
