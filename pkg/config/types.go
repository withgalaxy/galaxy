package config

type OutputType string

const (
	OutputStatic OutputType = "static"
	OutputServer OutputType = "server"
	OutputHybrid OutputType = "hybrid"
)

type AdapterName string

const (
	AdapterStandalone AdapterName = "standalone"
	AdapterCloudflare AdapterName = "cloudflare"
	AdapterNetlify    AdapterName = "netlify"
	AdapterVercel     AdapterName = "vercel"
)

type Config struct {
	Site           string          `toml:"site"`
	Base           string          `toml:"base"`
	OutDir         string          `toml:"outDir"`
	SrcDir         string          `toml:"srcDir"`
	PackageManager string          `toml:"packageManager"`
	Output         OutputConfig    `toml:"output"`
	Server         ServerConfig    `toml:"server"`
	Adapter        AdapterConfig   `toml:"adapter"`
	Security       SecurityConfig  `toml:"security"`
	Lifecycle      LifecycleConfig `toml:"lifecycle"`
	Plugins        []PluginConfig  `toml:"plugins"`
	Markdown       MarkdownConfig  `toml:"markdown"`
	Content        ContentConfig   `toml:"content"`
}

type OutputConfig struct {
	Type OutputType `toml:"type"`
}

type ServerConfig struct {
	Port int    `toml:"port"`
	Host string `toml:"host"`
}

type AdapterConfig struct {
	Name   AdapterName            `toml:"name"`
	Config map[string]interface{} `toml:"config"`
}

type LifecycleConfig struct {
	Enabled         bool `toml:"enabled"`
	StartupTimeout  int  `toml:"startupTimeout"`
	ShutdownTimeout int  `toml:"shutdownTimeout"`
}

type PluginConfig struct {
	Name   string                 `toml:"name"`
	Config map[string]interface{} `toml:"config"`
}

type MarkdownConfig struct {
	SyntaxHighlight string   `toml:"syntaxHighlight"`
	RemarkPlugins   []string `toml:"remarkPlugins"`
	RehypePlugins   []string `toml:"rehypePlugins"`
}

type ContentConfig struct {
	Collections bool   `toml:"collections"`
	ContentDir  string `toml:"contentDir"`
}

type SecurityConfig struct {
	CheckOrigin    bool            `toml:"checkOrigin"`
	AllowOrigins   []string        `toml:"allowOrigins"`
	AllowedDomains []RemotePattern `toml:"allowedDomains"`
	Headers        HeadersConfig   `toml:"headers"`
	BodyLimit      BodyLimitConfig `toml:"bodyLimit"`
}

type HeadersConfig struct {
	Enabled                 bool   `toml:"enabled"`
	XFrameOptions           string `toml:"xFrameOptions"`
	XContentTypeOptions     string `toml:"xContentTypeOptions"`
	XXSSProtection          string `toml:"xXSSProtection"`
	ReferrerPolicy          string `toml:"referrerPolicy"`
	StrictTransportSecurity string `toml:"strictTransportSecurity"`
	ContentSecurityPolicy   string `toml:"contentSecurityPolicy"`
	PermissionsPolicy       string `toml:"permissionsPolicy"`
}

type BodyLimitConfig struct {
	Enabled  bool  `toml:"enabled"`
	MaxBytes int64 `toml:"maxBytes"`
}

type RemotePattern struct {
	Protocol string `toml:"protocol"`
	Hostname string `toml:"hostname"`
	Port     *int   `toml:"port"`
}

func DefaultConfig() *Config {
	return &Config{
		Site:           "",
		Base:           "/",
		OutDir:         "./dist",
		SrcDir:         "./src",
		PackageManager: "npm",
		Output: OutputConfig{
			Type: OutputStatic,
		},
		Server: ServerConfig{
			Port: 4322,
			Host: "localhost",
		},
		Adapter: AdapterConfig{
			Name:   AdapterStandalone,
			Config: make(map[string]interface{}),
		},
		Lifecycle: LifecycleConfig{
			Enabled:         true,
			StartupTimeout:  30,
			ShutdownTimeout: 10,
		},
		Markdown: MarkdownConfig{
			SyntaxHighlight: "monokai",
			RemarkPlugins:   []string{},
			RehypePlugins:   []string{},
		},
		Content: ContentConfig{
			Collections: true,
			ContentDir:  "./src/content",
		},
		Security: SecurityConfig{
			CheckOrigin:    true,
			AllowOrigins:   []string{},
			AllowedDomains: []RemotePattern{},
			Headers: HeadersConfig{
				Enabled:                 false,
				XFrameOptions:           "DENY",
				XContentTypeOptions:     "nosniff",
				XXSSProtection:          "1; mode=block",
				ReferrerPolicy:          "strict-origin-when-cross-origin",
				StrictTransportSecurity: "max-age=31536000; includeSubDomains",
				ContentSecurityPolicy:   "",
				PermissionsPolicy:       "",
			},
			BodyLimit: BodyLimitConfig{
				Enabled:  true,
				MaxBytes: 10 << 20,
			},
		},
	}
}
