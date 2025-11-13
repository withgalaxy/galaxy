package netlify

import (
	"os"
)

func generateRedirects(path string) error {
	content := `# SPA fallback for client-side routing
/*    /index.html   200
`

	return os.WriteFile(path, []byte(content), 0644)
}
