package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/withgalaxy/galaxy/pkg/assets"
	"github.com/withgalaxy/galaxy/pkg/codegen"
	"github.com/withgalaxy/galaxy/pkg/config"
	"github.com/withgalaxy/galaxy/pkg/parser"
	"github.com/withgalaxy/galaxy/pkg/plugins"
	"github.com/withgalaxy/galaxy/pkg/plugins/tailwind"
	"github.com/withgalaxy/galaxy/pkg/router"
	"github.com/withgalaxy/galaxy/pkg/wasm"
)

type SSRBuilder struct {
	Config        *config.Config
	SrcDir        string
	PagesDir      string
	OutDir        string
	PublicDir     string
	Router        *router.Router
	PluginManager *plugins.Manager
	Bundler       *assets.Bundler
}

func NewSSRBuilder(cfg *config.Config, srcDir, pagesDir, outDir, publicDir string) *SSRBuilder {
	pluginMgr := plugins.NewManager(cfg)
	pluginMgr.Register(tailwind.New())

	bundler := assets.NewBundler(outDir)
	bundler.PluginManager = pluginMgr

	return &SSRBuilder{
		Config:        cfg,
		SrcDir:        srcDir,
		PagesDir:      pagesDir,
		OutDir:        outDir,
		PublicDir:     publicDir,
		Router:        router.NewRouter(pagesDir),
		PluginManager: pluginMgr,
		Bundler:       bundler,
	}
}

func (b *SSRBuilder) Build() error {
	baseDir := b.SrcDir
	if err := b.PluginManager.Load(baseDir, b.OutDir); err != nil {
		return fmt.Errorf("load plugins: %w", err)
	}

	buildCtx := &plugins.BuildContext{
		Config:    b.Config,
		RootDir:   baseDir,
		OutDir:    b.OutDir,
		PagesDir:  b.PagesDir,
		PublicDir: b.PublicDir,
	}

	if err := b.PluginManager.BuildStart(buildCtx); err != nil {
		return fmt.Errorf("plugin BuildStart: %w", err)
	}

	if err := b.Router.Discover(); err != nil {
		return fmt.Errorf("route discovery: %w", err)
	}
	b.Router.Sort()

	if err := os.RemoveAll(b.OutDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clean output: %w", err)
	}

	if err := os.MkdirAll(b.OutDir, 0755); err != nil {
		return fmt.Errorf("create output: %w", err)
	}

	serverDir := filepath.Join(b.OutDir, "server")
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		return fmt.Errorf("create server dir: %w", err)
	}

	if err := b.precompileWasmScripts(); err != nil {
		return fmt.Errorf("precompile wasm: %w", err)
	}

	if err := b.copyWasmAssets(); err != nil {
		return fmt.Errorf("copy wasm assets: %w", err)
	}

	if err := b.copyWasmExec(); err != nil {
		return fmt.Errorf("copy wasm exec: %w", err)
	}

	if err := b.generateServerCode(serverDir); err != nil {
		return fmt.Errorf("generate server: %w", err)
	}

	if err := b.copyPublicAssets(); err != nil {
		return fmt.Errorf("copy assets: %w", err)
	}

	if err := b.compileServer(serverDir); err != nil {
		return fmt.Errorf("compile server: %w", err)
	}

	if err := b.PluginManager.BuildEnd(buildCtx); err != nil {
		return fmt.Errorf("plugin BuildEnd: %w", err)
	}

	return nil
}

func (b *SSRBuilder) generateServerCode(serverDir string) error {
	moduleName, err := detectModuleName()
	if err != nil {
		moduleName = "generated-server"
	}

	codegenBuilder := codegen.NewCodegenBuilder(b.Router.Routes, b.PagesDir, b.OutDir, moduleName, b.PublicDir)
	return codegenBuilder.Build()
}

func (b *SSRBuilder) copyPublicAssets() error {
	publicOutDir := filepath.Join(b.OutDir, "public")

	if _, err := os.Stat(b.PublicDir); os.IsNotExist(err) {
		return nil
	}

	if err := os.MkdirAll(publicOutDir, 0755); err != nil {
		return err
	}

	return filepath.Walk(b.PublicDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(b.PublicDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(publicOutDir, relPath)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(destPath, data, info.Mode())
	})
}

func (b *SSRBuilder) compileServer(serverDir string) error {
	return nil
}

func detectModuleName() (string, error) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}

	return "", fmt.Errorf("module name not found")
}

func (b *SSRBuilder) precompileWasmScripts() error {
	manifest := wasm.NewManifest()

	for _, route := range b.Router.Routes {
		if route.IsEndpoint {
			continue
		}

		content, err := os.ReadFile(route.FilePath)
		if err != nil {
			continue
		}

		comp, err := parser.Parse(string(content))
		if err != nil {
			continue
		}

		wasmAssets, err := b.Bundler.BundleWasmScripts(comp, route.FilePath)
		if err != nil {
			return err
		}

		jsPath, err := b.Bundler.BundleScripts(comp, route.FilePath)
		if err != nil {
			return err
		}

		if len(wasmAssets) == 0 && jsPath == "" {
			continue
		}

		pageAssets := wasm.WasmPageAssets{}
		for _, asset := range wasmAssets {
			hash := extractHash(asset.LoaderPath)
			pageAssets.WasmModules = append(pageAssets.WasmModules, wasm.WasmModule{
				Hash:       hash,
				WasmPath:   asset.WasmPath,
				LoaderPath: asset.LoaderPath,
			})
		}
		if jsPath != "" {
			pageAssets.JSScripts = append(pageAssets.JSScripts, jsPath)
		}

		relPath, err := filepath.Rel(b.PagesDir, route.FilePath)
		if err != nil {
			relPath = route.FilePath
		}
		manifestKey := "pages/" + relPath
		manifest.Assets[manifestKey] = pageAssets
	}

	manifestPath := filepath.Join(b.OutDir, "server", "_assets", "wasm-manifest.json")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		return err
	}

	return manifest.Save(manifestPath)
}

func extractHash(loaderPath string) string {
	parts := strings.Split(filepath.Base(loaderPath), "-")
	if len(parts) >= 2 {
		return strings.TrimSuffix(parts[1], "-loader.js")
	}
	return ""
}

func (b *SSRBuilder) copyWasmAssets() error {
	assetsDir := filepath.Join(b.OutDir, "_assets")
	if _, err := os.Stat(assetsDir); os.IsNotExist(err) {
		return nil
	}

	serverAssetsDir := filepath.Join(b.OutDir, "server", "_assets")
	if err := os.MkdirAll(serverAssetsDir, 0755); err != nil {
		return err
	}

	return filepath.Walk(assetsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(assetsDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(serverAssetsDir, relPath)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(destPath, data, 0644)
	})
}

func (b *SSRBuilder) copyWasmExec() error {
	goRoot := os.Getenv("GOROOT")
	if goRoot == "" {
		cmd := exec.Command("go", "env", "GOROOT")
		output, err := cmd.Output()
		if err != nil {
			return nil
		}
		goRoot = strings.TrimSpace(string(output))
	}

	if goRoot == "" {
		return nil
	}

	wasmExecSrc := filepath.Join(goRoot, "misc", "wasm", "wasm_exec.js")
	if _, err := os.Stat(wasmExecSrc); os.IsNotExist(err) {
		wasmExecSrc = filepath.Join(goRoot, "lib", "wasm", "wasm_exec.js")
		if _, err := os.Stat(wasmExecSrc); os.IsNotExist(err) {
			return nil
		}
	}

	data, err := os.ReadFile(wasmExecSrc)
	if err != nil {
		return nil
	}

	wasmExecDest := filepath.Join(b.OutDir, "server", "wasm_exec.js")
	return os.WriteFile(wasmExecDest, data, 0644)
}
