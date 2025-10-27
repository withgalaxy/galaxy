package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/withgalaxy/galaxy/pkg/config"
	"github.com/withgalaxy/galaxy/pkg/parser"
	"github.com/spf13/cobra"
)

var (
	checkWatch bool
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check your project for errors",
	Long:  `Run diagnostics on your project and report errors`,
	RunE:  runCheck,
}

func init() {
	rootCmd.AddCommand(checkCmd)
	checkCmd.Flags().BoolVar(&checkWatch, "watch", false, "watch for changes")
}

func runCheck(cmd *cobra.Command, args []string) error {
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
	componentsDir := filepath.Join(srcDir, "components")

	if !silent {
		fmt.Println("🔍 Checking project...")
	}

	errors := 0
	warnings := 0

	if err = checkDirectory(pagesDir, &errors, &warnings); err != nil {
		return err
	}

	if _, err = os.Stat(componentsDir); err == nil {
		if err = checkDirectory(componentsDir, &errors, &warnings); err != nil {
			return err
		}
	}

	if !silent {
		fmt.Printf("\n")
		if errors > 0 {
			fmt.Printf("❌ Found %d error(s) and %d warning(s)\n", errors, warnings)
			os.Exit(1)
		} else if warnings > 0 {
			fmt.Printf("⚠️  Found %d warning(s)\n", warnings)
		} else {
			fmt.Printf("✅ No errors found\n")
		}
	}

	return nil
}

func checkDirectory(dir string, errors, warnings *int) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || filepath.Ext(path) != ".gxc" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		_, err = parser.Parse(string(content))
		if err != nil {
			*errors++
			if !silent {
				relPath, _ := filepath.Rel(filepath.Dir(dir), path)
				fmt.Printf("❌ %s: %v\n", relPath, err)
			}
		}

		return nil
	})
}
