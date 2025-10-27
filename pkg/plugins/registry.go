package plugins

import (
	"github.com/withgalaxy/galaxy/pkg/config"
)

func NewDefaultManager(cfg *config.Config) *Manager {
	mgr := NewManager(cfg)
	return mgr
}
