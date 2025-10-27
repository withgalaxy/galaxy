package cli

import (
	"fmt"
	"path/filepath"

	"github.com/withgalaxy/galaxy/pkg/prompts"
	"github.com/withgalaxy/galaxy/pkg/templates"
	"github.com/spf13/cobra"
)

var (
	noInteractive bool
)

var createCmd = &cobra.Command{
	Use:   "create [project-name]",
	Short: "Create a new Galaxy project",
	Long:  `Create a new Galaxy project with interactive setup`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runCreate,
}

func init() {
	createCmd.Flags().BoolVar(&noInteractive, "no-interactive", false, "Skip prompts and use defaults (pnpm, minimal template, yes to all)")
	rootCmd.AddCommand(createCmd)
}

func runCreate(cmd *cobra.Command, args []string) error {
	defaultName := "my-galaxy-project"
	if len(args) > 0 {
		defaultName = args[0]
	}

	var config *prompts.ProjectConfig
	var err error

	if noInteractive {
		config = prompts.GetDefaultConfig(defaultName)
	} else {
		fmt.Println("🚀 Welcome to Galaxy!")
		fmt.Println("Let's create your new project")

		config, err = prompts.AskProjectDetails(defaultName)
		if err != nil {
			return err
		}
	}

	projectPath := filepath.Join(".", config.Name)

	fmt.Printf("\n✨ Creating project in %s...\n", projectPath)

	gen, err := templates.NewGenerator(config.Template, config.Name, config.PackageManager)
	if err != nil {
		return err
	}

	if err := gen.Generate(projectPath); err != nil {
		return fmt.Errorf("failed to generate project: %w", err)
	}

	fmt.Println("✓ Project files created")

	if config.InitGit {
		fmt.Println("Initializing git repository...")
		if err := templates.InitGit(projectPath); err != nil {
			fmt.Printf("⚠ Failed to initialize git: %v\n", err)
		} else {
			fmt.Println("✓ Git initialized")
		}
	}

	if config.InstallDeps {
		fmt.Printf("Installing dependencies with %s...\n", config.PackageManager)
		if err := templates.InstallDependencies(projectPath, config.PackageManager); err != nil {
			fmt.Printf("⚠ Failed to install dependencies: %v\n", err)
		} else {
			fmt.Println("✓ Dependencies installed")
		}
	}

	fmt.Println("\n✅ Project created successfully!")
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  cd %s\n", config.Name)
	if !config.InstallDeps {
		fmt.Printf("  %s install\n", config.PackageManager)
	}
	fmt.Printf("  galaxy dev\n")

	return nil
}
