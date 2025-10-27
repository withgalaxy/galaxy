package plugins

import (
	"fmt"

	"github.com/withgalaxy/galaxy/pkg/config"
)

type Manager struct {
	registry map[string]Plugin
	plugins  []Plugin
	config   *config.Config
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		registry: make(map[string]Plugin),
		plugins:  []Plugin{},
		config:   cfg,
	}
}

func (m *Manager) Register(plugin Plugin) {
	m.registry[plugin.Name()] = plugin
}

func (m *Manager) Load(rootDir, outDir string) error {
	for _, pluginCfg := range m.config.Plugins {
		plugin, exists := m.registry[pluginCfg.Name]
		if !exists {
			return fmt.Errorf("unknown plugin: %s", pluginCfg.Name)
		}

		ctx := &SetupContext{
			Config:    m.config,
			PluginCfg: pluginCfg.Config,
			RootDir:   rootDir,
			OutDir:    outDir,
		}

		if err := plugin.Setup(ctx); err != nil {
			return fmt.Errorf("setup plugin %s: %w", pluginCfg.Name, err)
		}

		m.plugins = append(m.plugins, plugin)
	}

	return nil
}

func (m *Manager) TransformCSS(css string, filePath string) (string, error) {
	result := css
	for _, plugin := range m.plugins {
		transformed, err := plugin.TransformCSS(result, filePath)
		if err != nil {
			return "", fmt.Errorf("plugin %s: %w", plugin.Name(), err)
		}
		if transformed != "" {
			result = transformed
		}
	}
	return result, nil
}

func (m *Manager) TransformJS(js string, filePath string) (string, error) {
	result := js
	for _, plugin := range m.plugins {
		transformed, err := plugin.TransformJS(result, filePath)
		if err != nil {
			return "", fmt.Errorf("plugin %s: %w", plugin.Name(), err)
		}
		if transformed != "" {
			result = transformed
		}
	}
	return result, nil
}

func (m *Manager) InjectTags() []HTMLTag {
	var tags []HTMLTag
	for _, plugin := range m.plugins {
		tags = append(tags, plugin.InjectTags()...)
	}
	return tags
}

func (m *Manager) BuildStart(ctx *BuildContext) error {
	for _, plugin := range m.plugins {
		if err := plugin.BuildStart(ctx); err != nil {
			return fmt.Errorf("plugin %s BuildStart: %w", plugin.Name(), err)
		}
	}
	return nil
}

func (m *Manager) BuildEnd(ctx *BuildContext) error {
	for _, plugin := range m.plugins {
		if err := plugin.BuildEnd(ctx); err != nil {
			return fmt.Errorf("plugin %s BuildEnd: %w", plugin.Name(), err)
		}
	}
	return nil
}

func (m *Manager) Plugins() []Plugin {
	return m.plugins
}
