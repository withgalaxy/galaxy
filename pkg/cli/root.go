package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	Version = "0.15.0"
	cfgFile string
	rootDir string
	verbose bool
	silent  bool
)

var rootCmd = &cobra.Command{
	Use:   "galaxy",
	Short: "Galaxy - A blazing fast web framework CLI",
	Long: `Galaxy is a Go-based web framework inspired by Astro.
Build fast, content-focused websites with ease.

Complete documentation is available at https://github.com/yourusername/astro-go`,
	Version: Version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path")
	rootCmd.PersistentFlags().StringVar(&rootDir, "root", "", "project root directory")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable verbose logging")
	rootCmd.PersistentFlags().BoolVar(&silent, "silent", false, "disable all logging")
}
