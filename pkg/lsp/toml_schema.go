package lsp

import (
	"github.com/withgalaxy/galaxy/pkg/config"
)

type TOMLSchema struct {
	Tables map[string]TableSchema
}

type TableSchema struct {
	Fields      map[string]FieldSchema
	Description string
}

type FieldSchema struct {
	Type        string
	EnumValues  []string
	Default     string
	Description string
	Required    bool
	IsTable     bool
}

func BuildTOMLSchema() *TOMLSchema {
	return &TOMLSchema{
		Tables: map[string]TableSchema{
			"": {
				Fields: map[string]FieldSchema{
					"site": {
						Type:        "string",
						Description: "Base URL of your site (e.g., https://example.com)",
						Default:     "",
					},
					"base": {
						Type:        "string",
						Description: "Base path for your site (e.g., /docs/)",
						Default:     "/",
					},
					"outDir": {
						Type:        "string",
						Description: "Output directory for built files",
						Default:     "./dist",
					},
					"srcDir": {
						Type:        "string",
						Description: "Source directory for your project",
						Default:     "./src",
					},
					"packageManager": {
						Type:        "string",
						EnumValues:  []string{"npm", "yarn", "pnpm", "bun"},
						Description: "Package manager to use",
						Default:     "npm",
					},
					"output": {
						Type:        "table",
						Description: "Output configuration",
						IsTable:     true,
					},
					"server": {
						Type:        "table",
						Description: "Development server configuration",
						IsTable:     true,
					},
					"adapter": {
						Type:        "table",
						Description: "Deployment adapter configuration",
						IsTable:     true,
					},
					"security": {
						Type:        "table",
						Description: "Security configuration",
						IsTable:     true,
					},
					"lifecycle": {
						Type:        "table",
						Description: "Lifecycle hooks configuration",
						IsTable:     true,
					},
					"markdown": {
						Type:        "table",
						Description: "Markdown processing configuration",
						IsTable:     true,
					},
					"content": {
						Type:        "table",
						Description: "Content collections configuration",
						IsTable:     true,
					},
					"plugins": {
						Type:        "table",
						Description: "Plugin configuration (use [[plugins]] for array)",
						IsTable:     true,
					},
				},
			},
			"output": {
				Description: "Configure output type for your application",
				Fields: map[string]FieldSchema{
					"type": {
						Type:        "string",
						EnumValues:  []string{string(config.OutputStatic), string(config.OutputServer), string(config.OutputHybrid)},
						Description: "Rendering mode: static (SSG), server (SSR), or hybrid",
						Default:     string(config.OutputStatic),
					},
				},
			},
			"server": {
				Description: "Development server settings",
				Fields: map[string]FieldSchema{
					"port": {
						Type:        "int",
						Description: "Port number for dev server",
						Default:     "4322",
					},
					"host": {
						Type:        "string",
						Description: "Host address for dev server",
						Default:     "localhost",
					},
				},
			},
			"adapter": {
				Description: "Deployment adapter (required for server/hybrid output)",
				Fields: map[string]FieldSchema{
					"name": {
						Type: "string",
						EnumValues: []string{
							string(config.AdapterStandalone),
							string(config.AdapterCloudflare),
							string(config.AdapterNetlify),
							string(config.AdapterVercel),
						},
						Description: "Deployment target (vercel/netlify/cloudflare only support static)",
						Default:     string(config.AdapterStandalone),
					},
					"config": {
						Type:        "table",
						Description: "Adapter-specific configuration",
						IsTable:     true,
					},
				},
			},
			"lifecycle": {
				Description: "Lifecycle hooks configuration",
				Fields: map[string]FieldSchema{
					"enabled": {
						Type:        "bool",
						Description: "Enable lifecycle hooks",
						Default:     "true",
					},
					"startupTimeout": {
						Type:        "int",
						Description: "Startup timeout in seconds",
						Default:     "30",
					},
					"shutdownTimeout": {
						Type:        "int",
						Description: "Shutdown timeout in seconds",
						Default:     "10",
					},
				},
			},
			"markdown": {
				Description: "Markdown processing options",
				Fields: map[string]FieldSchema{
					"syntaxHighlight": {
						Type:        "string",
						Description: "Syntax highlighting theme",
						Default:     "monokai",
					},
					"remarkPlugins": {
						Type:        "array",
						Description: "Remark plugins to use",
					},
					"rehypePlugins": {
						Type:        "array",
						Description: "Rehype plugins to use",
					},
				},
			},
			"content": {
				Description: "Content collections configuration",
				Fields: map[string]FieldSchema{
					"collections": {
						Type:        "bool",
						Description: "Enable content collections",
						Default:     "true",
					},
					"contentDir": {
						Type:        "string",
						Description: "Content directory path",
						Default:     "./src/content",
					},
				},
			},
			"security": {
				Description: "Security settings",
				Fields: map[string]FieldSchema{
					"checkOrigin": {
						Type:        "bool",
						Description: "Enable origin checking",
						Default:     "true",
					},
					"allowOrigins": {
						Type:        "array",
						Description: "Allowed origins list",
					},
					"headers": {
						Type:        "table",
						Description: "Security headers configuration",
						IsTable:     true,
					},
					"bodyLimit": {
						Type:        "table",
						Description: "Request body size limits",
						IsTable:     true,
					},
				},
			},
			"security.headers": {
				Description: "Security headers configuration",
				Fields: map[string]FieldSchema{
					"enabled": {
						Type:        "bool",
						Description: "Enable security headers",
						Default:     "false",
					},
					"xFrameOptions": {
						Type:        "string",
						EnumValues:  []string{"DENY", "SAMEORIGIN"},
						Description: "X-Frame-Options header",
						Default:     "DENY",
					},
					"xContentTypeOptions": {
						Type:        "string",
						Description: "X-Content-Type-Options header",
						Default:     "nosniff",
					},
					"xXSSProtection": {
						Type:        "string",
						Description: "X-XSS-Protection header",
						Default:     "1; mode=block",
					},
					"referrerPolicy": {
						Type:        "string",
						Description: "Referrer-Policy header",
						Default:     "strict-origin-when-cross-origin",
					},
					"strictTransportSecurity": {
						Type:        "string",
						Description: "Strict-Transport-Security header",
						Default:     "max-age=31536000; includeSubDomains",
					},
					"contentSecurityPolicy": {
						Type:        "string",
						Description: "Content-Security-Policy header",
					},
					"permissionsPolicy": {
						Type:        "string",
						Description: "Permissions-Policy header",
					},
				},
			},
			"security.bodyLimit": {
				Description: "Request body size configuration",
				Fields: map[string]FieldSchema{
					"enabled": {
						Type:        "bool",
						Description: "Enable body size limit",
						Default:     "true",
					},
					"maxBytes": {
						Type:        "int",
						Description: "Maximum body size in bytes",
						Default:     "10485760",
					},
				},
			},
		},
	}
}

func (s *TOMLSchema) GetTableSchema(table string) (TableSchema, bool) {
	schema, ok := s.Tables[table]
	return schema, ok
}

func (s *TOMLSchema) GetFieldSchema(table, field string) (FieldSchema, bool) {
	tableSchema, ok := s.Tables[table]
	if !ok {
		return FieldSchema{}, false
	}
	fieldSchema, ok := tableSchema.Fields[field]
	return fieldSchema, ok
}
