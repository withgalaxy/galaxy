package netlify

import (
	"os"

	"github.com/withgalaxy/galaxy/pkg/adapters"
)

func generateHeaders(path string, cfg *adapters.BuildConfig) error {
	content := `# Cache-control for hashed assets
/_assets/*
  Cache-Control: public, max-age=31536000, immutable
`

	return os.WriteFile(path, []byte(content), 0644)
}
