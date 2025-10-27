package adapters

import "github.com/withgalaxy/galaxy/pkg/config"

type Adapter interface {
	Name() string
	Build(cfg *BuildConfig) error
}

type BuildConfig struct {
	Config    *config.Config
	ServerDir string
	OutDir    string
	PagesDir  string
	PublicDir string
	Routes    []RouteInfo
}

type RouteInfo struct {
	Pattern    string
	FilePath   string
	IsEndpoint bool
}
