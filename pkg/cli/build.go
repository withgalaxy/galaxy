package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/withgalaxy/galaxy/pkg/build"
	"github.com/withgalaxy/galaxy/pkg/config"
	"github.com/spf13/cobra"
)

var (
	buildOutDir string
	buildMode   string
	buildForce  bool
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build your project for production",
	Long:  `Build your project and write it to disk`,
	RunE:  runBuild,
}

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().StringVar(&buildOutDir, "outDir", "./dist", "output directory")
	buildCmd.Flags().StringVar(&buildMode, "mode", "production", "build mode")
	buildCmd.Flags().BoolVar(&buildForce, "force", false, "clear cache and rebuild")
}

func runBuild(cmd *cobra.Command, args []string) error {
	start := time.Now()

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	if rootDir != "" {
		cwd = rootDir
	}

	cfg, err := config.LoadFromDir(cwd)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	srcDir := cfg.SrcDir
	if !filepath.IsAbs(srcDir) {
		srcDir = filepath.Join(cwd, srcDir)
	}

	pagesDir := filepath.Join(srcDir, "pages")
	publicDir := filepath.Join(cwd, "public")
	outDir := buildOutDir
	if !filepath.IsAbs(outDir) {
		outDir = filepath.Join(cwd, outDir)
	}

	if _, err = os.Stat(pagesDir); os.IsNotExist(err) {
		return fmt.Errorf("pages directory not found: %s", pagesDir)
	}

	if buildOutDir != "./dist" {
		cfg.OutDir = buildOutDir
	}

	if !filepath.IsAbs(cfg.OutDir) {
		cfg.OutDir = filepath.Join(cwd, cfg.OutDir)
	}
	outDir = cfg.OutDir

	if !silent {
		outputType := cfg.Output.Type
		if outputType == "" {
			outputType = config.OutputStatic
		}
		fmt.Printf("🔨 Building for production (%s mode)...\n", outputType)
		if verbose {
			fmt.Printf("📁 Pages: %s\n", pagesDir)
			fmt.Printf("📦 Public: %s\n", publicDir)
			fmt.Printf("📤 Output: %s\n", outDir)
		}
		fmt.Println()
	}

	var buildErr error

	if cfg.IsStatic() {
		builder := build.NewSSGBuilder(cfg, srcDir, pagesDir, outDir, publicDir)
		buildErr = builder.Build()
	} else if cfg.IsHybrid() {
		builder := build.NewHybridBuilder(cfg, srcDir, pagesDir, outDir, publicDir)
		buildErr = builder.Build()
	} else if cfg.IsSSR() {
		builder := build.NewSSRBuilder(cfg, srcDir, pagesDir, outDir, publicDir)
		buildErr = builder.Build()
	} else {
		return fmt.Errorf("unsupported output type: %s", cfg.Output.Type)
	}

	if buildErr != nil {
		return fmt.Errorf("build failed: %w", buildErr)
	}

	duration := time.Since(start)

	if !silent {
		fmt.Printf("\n✅ Build complete in %v\n", duration.Round(time.Millisecond))
		fmt.Printf("📂 Output: %s\n", outDir)
	}

	return nil
}
