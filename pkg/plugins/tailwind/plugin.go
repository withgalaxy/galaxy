package tailwind

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/withgalaxy/galaxy/pkg/plugins"
)

type TailwindPlugin struct {
	setupCtx   *plugins.SetupContext
	configPath string
	enabled    bool
}

func New() *TailwindPlugin {
	return &TailwindPlugin{}
}

func (p *TailwindPlugin) Name() string {
	return "tailwindcss"
}

func (p *TailwindPlugin) Setup(ctx *plugins.SetupContext) error {
	p.setupCtx = ctx

	configPath := filepath.Join(ctx.RootDir, "tailwind.config.js")
	if cfgPath, ok := ctx.PluginCfg["config"].(string); ok {
		configPath = filepath.Join(ctx.RootDir, cfgPath)
	}

	if _, err := os.Stat(configPath); err == nil {
		p.configPath = configPath
		p.enabled = true
	} else {
		p.enabled = false
		fmt.Printf("  ⚠ Tailwind config not found at %s, plugin disabled\n", configPath)
	}

	return nil
}

func (p *TailwindPlugin) TransformCSS(css string, filePath string) (string, error) {
	if !p.enabled {
		return css, nil
	}

	hasTailwindV3 := strings.Contains(css, "@tailwind")
	hasTailwindV4 := strings.Contains(css, `@import "tailwindcss"`) || strings.Contains(css, `@import 'tailwindcss'`)

	if !hasTailwindV3 && !hasTailwindV4 {
		return css, nil
	}

	tmpInput := filepath.Join(p.setupCtx.OutDir, ".tailwind-input.css")
	tmpOutput := filepath.Join(p.setupCtx.OutDir, ".tailwind-output.css")

	if err := os.WriteFile(tmpInput, []byte(css), 0644); err != nil {
		return "", fmt.Errorf("write temp input: %w", err)
	}
	defer os.Remove(tmpInput)

	var cmd *exec.Cmd
	if hasTailwindV4 {
		cmd = exec.Command("npx", "@tailwindcss/cli",
			"-i", tmpInput,
			"-o", tmpOutput,
			"--config", p.configPath,
			"--minify",
		)
	} else {
		cmd = exec.Command("npx", "tailwindcss",
			"-i", tmpInput,
			"-o", tmpOutput,
			"--config", p.configPath,
			"--minify",
		)
	}
	cmd.Dir = p.setupCtx.RootDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if hasTailwindV4 {
			return "", fmt.Errorf("run @tailwindcss/cli: %w\nMake sure @tailwindcss/cli is installed: npm install -D @tailwindcss/cli\n%s", err, stderr.String())
		}
		return "", fmt.Errorf("run tailwindcss: %w\n%s", err, stderr.String())
	}
	defer os.Remove(tmpOutput)

	processed, err := os.ReadFile(tmpOutput)
	if err != nil {
		return "", fmt.Errorf("read processed css: %w", err)
	}

	return string(processed), nil
}

func (p *TailwindPlugin) TransformJS(js string, filePath string) (string, error) {
	return js, nil
}

func (p *TailwindPlugin) InjectTags() []plugins.HTMLTag {
	return nil
}

func (p *TailwindPlugin) BuildStart(ctx *plugins.BuildContext) error {
	if p.enabled {
		fmt.Println("  🎨 Tailwind CSS plugin enabled")
	}
	return nil
}

func (p *TailwindPlugin) BuildEnd(ctx *plugins.BuildContext) error {
	return nil
}
