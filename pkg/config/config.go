package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}

func LoadFromDir(dir string) (*Config, error) {
	configPath := filepath.Join(dir, "galaxy.config.toml")
	return Load(configPath)
}

func (c *Config) Validate() error {
	switch c.Output.Type {
	case OutputStatic, OutputServer, OutputHybrid:
	case "":
		c.Output.Type = OutputStatic
	default:
		return fmt.Errorf("invalid output type: %s (must be static, server, or hybrid)", c.Output.Type)
	}

	if c.Output.Type == OutputServer || c.Output.Type == OutputHybrid {
		if c.Adapter.Name == "" {
			c.Adapter.Name = AdapterStandalone
		}

		switch c.Adapter.Name {
		case AdapterStandalone, AdapterCloudflare, AdapterNetlify, AdapterVercel:
		default:
			return fmt.Errorf("invalid adapter: %s", c.Adapter.Name)
		}
	}

	if c.Adapter.Name == AdapterVercel && c.Output.Type != OutputStatic {
		return fmt.Errorf(`vercel adapter only supports static output

Current configuration:
  output.type = "%s"
  adapter.name = "vercel"

Vercel does not support Go-based serverless functions. To fix this:
  1. Change output.type = "static" in galaxy.config.toml, OR
  2. Use adapter.name = "standalone" for SSR/hybrid output

For SSR deployments, use the standalone adapter with platforms like Docker, Railway, or Fly.io`, c.Output.Type)
	}

	if c.Adapter.Name == AdapterNetlify && c.Output.Type != OutputStatic {
		return fmt.Errorf(`netlify adapter only supports static output

Current configuration:
  output.type = "%s"
  adapter.name = "netlify"

Netlify does not support Go-based serverless functions. To fix this:
  1. Change output.type = "static" in galaxy.config.toml, OR
  2. Use adapter.name = "standalone" for SSR/hybrid output

For SSR deployments, use the standalone adapter with platforms like Docker, Railway, or Fly.io`, c.Output.Type)
	}

	if c.Server.Port == 0 {
		c.Server.Port = 4322
	}

	if c.Server.Host == "" {
		c.Server.Host = "localhost"
	}

	if c.OutDir == "" {
		c.OutDir = "./dist"
	}

	if c.Base == "" {
		c.Base = "/"
	}

	if c.PackageManager == "" {
		c.PackageManager = "npm"
	}

	if c.SrcDir == "" {
		c.SrcDir = "./src"
	}

	return nil
}

func (c *Config) IsSSR() bool {
	return c.Output.Type == OutputServer || c.Output.Type == OutputHybrid
}

func (c *Config) IsStatic() bool {
	return c.Output.Type == OutputStatic
}

func (c *Config) IsHybrid() bool {
	return c.Output.Type == OutputHybrid
}
