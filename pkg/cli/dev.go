package cli

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/cameron-webmatter/galaxy/pkg/config"
	"github.com/cameron-webmatter/galaxy/pkg/server"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

var (
	devPort    int
	devHost    string
	devOpen    bool
	devVerbose bool
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Start the development server",
	Long:  `Start the development server with hot reload`,
	RunE:  runDev,
}

func init() {
	rootCmd.AddCommand(devCmd)
	devCmd.Flags().IntVar(&devPort, "port", 4322, "port to run server on")
	devCmd.Flags().StringVar(&devHost, "host", "localhost", "host to bind to")
	devCmd.Flags().BoolVar(&devOpen, "open", false, "open browser on start")
	devCmd.Flags().BoolVar(&devVerbose, "verbose", true, "enable request logging")
}

func runDev(cmd *cobra.Command, args []string) error {
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

	if _, err = os.Stat(pagesDir); os.IsNotExist(err) {
		return fmt.Errorf("pages directory not found: %s", pagesDir)
	}

	srv := server.NewDevServer(cfg, cwd, pagesDir, publicDir, devPort, devVerbose)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	if err := watcher.Add(srcDir); err != nil {
		return err
	}
	if err := addRecursive(watcher, srcDir); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) != 0 {
					if !verbose && !silent {
						fmt.Printf("🔄 Change detected: %s\n", filepath.Base(event.Name))
					}

					if event.Op&fsnotify.Create != 0 {
						info, err := os.Stat(event.Name)
						if err == nil && info.IsDir() && isUnderDir(event.Name, srcDir) {
							if err := watcher.Add(event.Name); err == nil {
								if err := addRecursive(watcher, event.Name); err != nil && !silent {
									fmt.Printf("⚠ Failed to watch new directory: %v\n", err)
								}
							}
						}
					}

					ext := filepath.Ext(event.Name)
					isMarkdown := ext == ".md" || ext == ".mdx"
					isGalaxy := ext == ".gxc"

					if (isGalaxy || isMarkdown) && isUnderDir(event.Name, srcDir) {
						srv.Compiler.ClearCache()

						isComponent := !isUnderDir(event.Name, pagesDir) && !isMarkdown

						if srv.ChangeTracker != nil && srv.HMRServer != nil {
							diff, err := srv.ChangeTracker.DetectChange(event.Name)
							if err == nil {
								if isComponent && srv.ComponentTracker != nil {
									affectedPages := srv.ComponentTracker.GetAffectedPages(event.Name)
									if len(affectedPages) > 0 {
										componentName := strings.TrimSuffix(filepath.Base(event.Name), ".gxc")
										srv.HMRServer.BroadcastComponentUpdate(event.Name, componentName)
										if !verbose && !silent {
											fmt.Printf("🧩 Component updated: %s (affects %d page(s))\n", componentName, len(affectedPages))
										}
									} else {
										srv.HMRServer.BroadcastReload()
									}
								} else {
									needsCodegenRebuild := false
									hasStyleChange := false
									hasTemplateChange := false
									hasWasmChange := false

									if diff.StylesChanged {
										hasStyleChange = true
										needsCodegenRebuild = true
									}

									if diff.ScriptsChanged || diff.FrontmatterChanged {
										hasWasmChange = true
										needsCodegenRebuild = true
									}

									if diff.TemplateChanged {
										hasTemplateChange = true
										needsCodegenRebuild = true
									}

									if needsCodegenRebuild && !isComponent && isUnderDir(event.Name, pagesDir) {
										changeTypes := ""
										if hasStyleChange {
											changeTypes += "style,"
										}
										if hasTemplateChange {
											changeTypes += "template,"
										}
										if hasWasmChange {
											changeTypes += "wasm,"
										}
										changeTypes = strings.TrimSuffix(changeTypes, ",")
										srv.ScheduleCodegenRebuildWithType(event.Name, changeTypes)
									}
								}
							}
						}

						if isUnderDir(event.Name, pagesDir) && event.Op&(fsnotify.Create|fsnotify.Remove) != 0 {
							if err := srv.ReloadRoutes(); err != nil && !silent {
								fmt.Printf("⚠ Failed to reload routes: %v\n", err)
							}
						}

						if isMarkdown && srv.HMRServer != nil {
							srv.HMRServer.BroadcastReload()
							if !verbose && !silent {
								fmt.Printf("📄 Markdown updated: %s\n", filepath.Base(event.Name))
							}
						}
					}
					if filepath.Base(event.Name) == "middleware.go" && isUnderDir(event.Name, srcDir) {
						if !verbose && !silent {
							fmt.Printf("🔄 Reloading middleware...\n")
						}
						if err := srv.ReloadMiddleware(); err != nil && !silent {

							if srv.HMRServer != nil {
								srv.HMRServer.BroadcastWasmReload(event.Name, "", filepath.Base(event.Name))
							}

							fmt.Printf("⚠ Middleware reload failed: %v\n", err)
						} else if !verbose && !silent {
							fmt.Printf("✅ Middleware reloaded\n")
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				if err != nil && !silent {
					fmt.Printf("⚠ Watcher error: %v\n", err)
				}
			}
		}
	}()

	go handleInput(srv)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n👋 Shutting down...")

		srv.Shutdown()

		if srv.Lifecycle != nil {
			if err := srv.Lifecycle.ExecuteShutdown(); err != nil {
				fmt.Printf("⚠ Shutdown error: %v\n", err)
			}
		}
		os.Exit(0)
	}()

	if devOpen {
		go openBrowser(fmt.Sprintf("http://%s:%d", devHost, devPort))
	}

	fmt.Println("\n⚡ Hotkeys:")
	fmt.Println("  o + enter  →  Open browser")
	fmt.Println("  r + enter  →  Restart server")
	fmt.Println("  c + enter  →  Clear console")
	fmt.Println("  q + enter  →  Quit")

	return srv.Start()
}

func addRecursive(watcher *fsnotify.Watcher, dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
}

func handleInput(srv *server.DevServer) {
	buf := make([]byte, 1)
	for {
		os.Stdin.Read(buf)
		switch buf[0] {
		case 'o':
			openBrowser(fmt.Sprintf("http://localhost:%d", srv.Port))
		case 'r':
			fmt.Println("🔄 Restart requested (not implemented)")
		case 'c':
			clearConsole()
		case 'q':
			fmt.Println("\n👋 Shutting down...")
			os.Exit(0)
		}
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		cmd.Start()
	}
}

func clearConsole() {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin", "linux":
		cmd = exec.Command("clear")
	case "windows":
		cmd = exec.Command("cmd", "/c", "cls")
	}
	if cmd != nil {
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

func isUnderDir(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..") && rel != "."
}
