package cloudflare

import "os"

func generateRedirects(redirectsPath string) error {
	content := `/* /index.html 200
`

	return os.WriteFile(redirectsPath, []byte(content), 0644)
}
