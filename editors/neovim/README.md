# GXC Language Support for Neovim

Language server support for `.gxc` files (Galaxy components) and `galaxy.config.toml` in Neovim.

## Features

- **Diagnostics**: Real-time error checking for Go syntax and undefined variables
- **Auto-completion**: Variables from frontmatter, galaxy directives, TOML configuration options
- **Hover Info**: Type information and values for variables
- **Go to Definition**: Jump to variable declarations in frontmatter
- **TOML Support**: Validation and completions for `galaxy.config.toml` files

## Prerequisites

1. Install Galaxy CLI:
```bash
go install github.com/withgalaxy/galaxy/cmd/galaxy@v0.46.0-alpha.2
```

2. Ensure `galaxy` is in your PATH:
```bash
which galaxy  # Should print the path to galaxy binary
```

3. Install `nvim-lspconfig` plugin (if not already installed)

## Configuration

### Using lazy.nvim

Add to your Neovim configuration:

```lua
return {
  {
    "neovim/nvim-lspconfig",
    config = function()
      local lspconfig = require("lspconfig")
      local configs = require("lspconfig.configs")

      -- Register gxc language server
      if not configs.gxc then
        configs.gxc = {
          default_config = {
            cmd = { "galaxy", "lsp-server" },
            filetypes = { "gxc" },
            root_dir = lspconfig.util.root_pattern("galaxy.config.toml", ".git"),
            settings = {},
          },
        }
      end

      -- Setup gxc LSP
      lspconfig.gxc.setup({
        on_attach = function(client, bufnr)
          -- Your on_attach configuration
        end,
        capabilities = require("cmp_nvim_lsp").default_capabilities(),
      })
    end,
  },
}
```

### Using Packer

```lua
use {
  "neovim/nvim-lspconfig",
  config = function()
    local lspconfig = require("lspconfig")
    local configs = require("lspconfig.configs")

    if not configs.gxc then
      configs.gxc = {
        default_config = {
          cmd = { "galaxy", "lsp-server" },
          filetypes = { "gxc" },
          root_dir = lspconfig.util.root_pattern("galaxy.config.toml", ".git"),
          settings = {},
        },
      }
    end

    lspconfig.gxc.setup({})
  end,
}
```

### Minimal Setup (init.lua)

```lua
local lspconfig = require("lspconfig")
local configs = require("lspconfig.configs")

-- Register gxc language server
if not configs.gxc then
  configs.gxc = {
    default_config = {
      cmd = { "galaxy", "lsp-server" },
      filetypes = { "gxc" },
      root_dir = lspconfig.util.root_pattern("galaxy.config.toml", ".git"),
      settings = {},
    },
  }
end

-- Setup gxc LSP
lspconfig.gxc.setup({})
```

## File Type Detection

Add to your Neovim configuration to detect `.gxc` files:

```lua
vim.filetype.add({
  extension = {
    gxc = "gxc",
  },
})
```

Or create `~/.config/nvim/ftdetect/gxc.vim`:

```vim
au BufRead,BufNewFile *.gxc set filetype=gxc
```

## Syntax Highlighting

For basic syntax highlighting, you can configure Treesitter or use a simple vim syntax file.

### Option 1: Use HTML syntax as fallback

Add to your configuration:

```lua
vim.api.nvim_create_autocmd({"BufRead", "BufNewFile"}, {
  pattern = "*.gxc",
  callback = function()
    vim.bo.filetype = "html"
  end,
})
```

### Option 2: Custom syntax file (basic)

Create `~/.config/nvim/syntax/gxc.vim`:

```vim
" Basic GXC syntax highlighting
if exists("b:current_syntax")
  finish
endif

" Load HTML syntax as base
runtime! syntax/html.vim
unlet b:current_syntax

" Go frontmatter
syntax region gxcFrontmatter start=/\%^---$/ end=/^---$/ contains=@goCode
syntax include @goCode syntax/go.vim

" Galaxy directives
syntax match gxcDirective /galaxy:\w\+/

highlight link gxcDirective Keyword
highlight link gxcFrontmatter Comment

let b:current_syntax = "gxc"
```

## TOML Configuration Support

The LSP also provides support for `galaxy.config.toml` files. To enable:

```lua
-- Add TOML files to gxc LSP
lspconfig.gxc.setup({
  filetypes = { "gxc", "toml" },
  root_dir = lspconfig.util.root_pattern("galaxy.config.toml", ".git"),
})
```

Or use a more specific pattern:

```lua
vim.api.nvim_create_autocmd({"BufRead", "BufNewFile"}, {
  pattern = "galaxy.config.toml",
  callback = function()
    vim.lsp.start({
      name = "gxc",
      cmd = { "galaxy", "lsp-server" },
      root_dir = vim.fs.dirname(vim.fs.find({"galaxy.config.toml"}, { upward = true })[1]),
    })
  end,
})
```

## Troubleshooting

### LSP not starting

1. Check if galaxy is installed and in PATH:
```bash
which galaxy
galaxy lsp-server --help
```

2. Check LSP logs in Neovim:
```vim
:LspLog
```

3. Verify file type is set correctly:
```vim
:set filetype?
```

### No completions or diagnostics

1. Ensure LSP client is attached:
```vim
:LspInfo
```

2. Check if the LSP server is running:
```vim
:lua print(vim.inspect(vim.lsp.get_active_clients()))
```

### Custom galaxy binary path

If `galaxy` is not in your PATH, specify the full path:

```lua
configs.gxc = {
  default_config = {
    cmd = { "/full/path/to/galaxy", "lsp-server" },
    -- ... rest of config
  },
}
```

## Example GXC File

```gxc
---
var title = "Hello World"
var items = []string{"A", "B", "C"}
---
<h1>{title}</h1>
<ul>
  <li galaxy:for={item in items}>{item}</li>
</ul>

<style scoped>
h1 { color: blue; }
</style>
```

## Additional Resources

- [Galaxy Documentation](https://github.com/withgalaxy/galaxy)
- [nvim-lspconfig](https://github.com/neovim/nvim-lspconfig)
- [LSP Configuration Guide](https://github.com/neovim/nvim-lspconfig/blob/master/doc/server_configurations.md)
