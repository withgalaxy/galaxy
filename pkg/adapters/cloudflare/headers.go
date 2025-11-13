package cloudflare

import (
	"os"

	"github.com/withgalaxy/galaxy/pkg/adapters"
)

func generateHeaders(headersPath string, cfg *adapters.BuildConfig) error {
	content := `/_assets/*
  Cache-Control: public, max-age=31536000, immutable
`

	return os.WriteFile(headersPath, []byte(content), 0644)
}
