package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/withgalaxy/galaxy/pkg/config"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display environment information",
	Long:  `Display useful information about your current Galaxy setup`,
	RunE:  runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func runInfo(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	if rootDir != "" {
		cwd = rootDir
	}

	fmt.Printf("Galaxy                   v%s\n", Version)
	fmt.Printf("Go                       %s\n", runtime.Version())
	fmt.Printf("System                   %s (%s)\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Working Directory        %s\n", cwd)

	cfg, err := config.LoadFromDir(cwd)
	if err == nil {
		srcDir := cfg.SrcDir
		if !filepath.IsAbs(srcDir) {
			srcDir = filepath.Join(cwd, srcDir)
		}

		configPath := filepath.Join(cwd, "galaxy.config.toml")
		if _, err := os.Stat(configPath); err == nil {
			fmt.Printf("Config                   %s\n", configPath)
		}

		pagesDir := filepath.Join(srcDir, "pages")
		if info, err := os.Stat(pagesDir); err == nil && info.IsDir() {
			fmt.Printf("Pages                    %s\n", pagesDir)
		}

		componentsDir := filepath.Join(srcDir, "components")
		if info, err := os.Stat(componentsDir); err == nil && info.IsDir() {
			fmt.Printf("Components               %s\n", componentsDir)
		}
	}

	publicDir := filepath.Join(cwd, "public")
	if info, err := os.Stat(publicDir); err == nil && info.IsDir() {
		fmt.Printf("Public                   %s\n", publicDir)
	}

	distDir := filepath.Join(cwd, "dist")
	if info, err := os.Stat(distDir); err == nil && info.IsDir() {
		fmt.Printf("Output                   %s\n", distDir)
	}

	return nil
}
