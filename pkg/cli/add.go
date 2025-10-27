package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/BurntSushi/toml"
	"github.com/withgalaxy/galaxy/pkg/config"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [integration]",
	Short: "Add an integration to your project",
	Long:  `Add integrations like frameworks, adapters, or features`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)
}

var availableIntegrations = []string{
	"react",
	"vue",
	"svelte",
	"tailwind",
	"sitemap",
}

func runAdd(cmd *cobra.Command, args []string) error {
	var integration string

	if len(args) > 0 {
		integration = args[0]
	} else {
		prompt := &survey.Select{
			Message: "Select an integration:",
			Options: availableIntegrations,
		}
		if err := survey.AskOne(prompt, &integration); err != nil {
			return err
		}
	}

	fmt.Printf("\n📦 Adding %s integration...\n", integration)

	switch integration {
	case "react", "vue", "svelte":
		return addFramework(integration)
	case "tailwind":
		return addTailwind()
	case "sitemap":
		return addSitemap()
	default:
		return fmt.Errorf("unknown integration: %s", integration)
	}
}

func addFramework(framework string) error {
	fmt.Printf("ℹ️  Framework integrations not yet implemented\n")
	fmt.Printf("   This would install %s support for component islands\n", framework)
	return nil
}

func addTailwind() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	if rootDir != "" {
		cwd = rootDir
	}

	configPath := filepath.Join(cwd, "galaxy.config.toml")
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	pkgManager := cfg.PackageManager
	if pkgManager == "" {
		pkgManager = detectPackageManager(cwd)
	}

	packageJSONPath := filepath.Join(cwd, "package.json")
	if _, err := os.Stat(packageJSONPath); os.IsNotExist(err) {
		fmt.Println("Creating package.json...")
		cmd := exec.Command(pkgManager, "init", "-y")
		cmd.Dir = cwd
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create package.json: %w", err)
		}
	}

	var version string
	versionPrompt := &survey.Select{
		Message: "Select Tailwind CSS version:",
		Options: []string{"v4 (latest, recommended)", "v3 (stable)"},
		Default: "v4 (latest, recommended)",
	}
	if err := survey.AskOne(versionPrompt, &version); err != nil {
		return err
	}

	fmt.Println("Installing Tailwind CSS...")

	var installPkg string
	if version[0] == 'v' && version[1] == '4' {
		installPkg = "@tailwindcss/cli@latest"
	} else {
		installPkg = "tailwindcss@^3"
	}

	var cmd *exec.Cmd
	switch pkgManager {
	case "npm":
		cmd = exec.Command(pkgManager, "install", "-D", installPkg)
	case "pnpm", "yarn", "bun":
		cmd = exec.Command(pkgManager, "add", "-D", installPkg)
	default:
		cmd = exec.Command(pkgManager, "install", "-D", installPkg)
	}
	cmd.Dir = cwd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	fmt.Println("Creating Tailwind config...")
	var configContent string
	if version[0] == 'v' && version[1] == '4' {
		configContent = `/** @type {import('tailwindcss').Config} */
export default {
  content: ['src/**/*.gxc'],
}
`
	} else {
		configContent = `/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['src/**/*.gxc'],
  theme: {
    extend: {},
  },
  plugins: [],
}
`
	}

	tailwindConfigPath := filepath.Join(cwd, "tailwind.config.js")
	if err := os.WriteFile(tailwindConfigPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to create tailwind.config.js: %w", err)
	}

	hasPlugin := false
	for _, p := range cfg.Plugins {
		if p.Name == "tailwindcss" {
			hasPlugin = true
			break
		}
	}

	if !hasPlugin {
		cfg.Plugins = append(cfg.Plugins, config.PluginConfig{
			Name:   "tailwindcss",
			Config: make(map[string]interface{}),
		})

		f, err := os.Create(configPath)
		if err != nil {
			return fmt.Errorf("open config: %w", err)
		}
		defer f.Close()

		if err := toml.NewEncoder(f).Encode(cfg); err != nil {
			return fmt.Errorf("write config: %w", err)
		}

		fmt.Println("  ✓ Added tailwindcss plugin to galaxy.config.toml")
	}

	fmt.Println("\n✅ Tailwind CSS added!")
	fmt.Println("\nNext steps:")
	if version[0] == 'v' && version[1] == '4' {
		fmt.Println("  1. Add '@import \"tailwindcss\";' to your CSS file")
		fmt.Println("  2. Link the CSS file in your Layout component")
	} else {
		fmt.Println("  1. Add Tailwind directives to your CSS:")
		fmt.Println("     @tailwind base;")
		fmt.Println("     @tailwind components;")
		fmt.Println("     @tailwind utilities;")
	}
	return nil
}

func detectPackageManager(cwd string) string {
	if _, err := os.Stat(filepath.Join(cwd, "pnpm-lock.yaml")); err == nil {
		return "pnpm"
	}
	if _, err := os.Stat(filepath.Join(cwd, "yarn.lock")); err == nil {
		return "yarn"
	}
	if _, err := os.Stat(filepath.Join(cwd, "bun.lockb")); err == nil {
		return "bun"
	}
	return "npm"
}

func addSitemap() error {
	fmt.Println("ℹ️  Sitemap integration not yet implemented")
	fmt.Println("   This would generate sitemap.xml during build")
	return nil
}
