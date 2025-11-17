# NeoVim Support for Galaxy (.gxc)

NeoVim setup for `.gxc` file syntax highlighting and LSP support.

## Prerequisites

- NeoVim 0.8+
- `galaxy` CLI installed and in PATH
- [nvim-lspconfig](https://github.com/neovim/nvim-lspconfig) plugin

## Installation

### Option 1: Manual Installation

1. Copy plugin files to your NeoVim config:
```bash
cp -r editors/nvim/* ~/.config/nvim/
```

2. Add LSP setup to your `init.lua`:
```lua
require'lspconfig'.gxc.setup{}
```

### Option 2: Using lazy.nvim

Add to your plugins:
```lua
{
  'neovim/nvim-lspconfig',
  config = function()
    local lspconfig = require('lspconfig')
    
    local configs = require('lspconfig.configs')
    if not configs.gxc then
      configs.gxc = {
        default_config = {
          cmd = { 'galaxy', 'lsp-server', '--stdio' },
          filetypes = { 'gxc' },
          root_dir = lspconfig.util.root_pattern('galaxy.config.json', 'galaxy.config.toml', '.git'),
          single_file_support = true,
        },
      }
    end
    
    lspconfig.gxc.setup{}
  end
}
```

Then add syntax files:
```bash
mkdir -p ~/.config/nvim/{ftdetect,syntax}
cp editors/nvim/ftdetect/gxc.vim ~/.config/nvim/ftdetect/
cp editors/nvim/syntax/gxc.vim ~/.config/nvim/syntax/
```

### Option 3: Using packer.nvim

```lua
use {
  'neovim/nvim-lspconfig',
  config = function()
    local lspconfig = require('lspconfig')
    local configs = require('lspconfig.configs')
    
    if not configs.gxc then
      configs.gxc = {
        default_config = {
          cmd = { 'galaxy', 'lsp-server', '--stdio' },
          filetypes = { 'gxc' },
          root_dir = lspconfig.util.root_pattern('galaxy.config.json', 'galaxy.config.toml', '.git'),
          single_file_support = true,
        },
      }
    end
    
    lspconfig.gxc.setup{}
  end
}
```

## Features

### Syntax Highlighting
- Frontmatter (Go code between `---`)
- HTML template syntax
- `<script>` tags (JavaScript)
- `<style>` tags (CSS)
- Template interpolation `{variable}`
- Galaxy directives (`galaxy:if`, `galaxy:for`, `galaxy:else`)

### LSP Features
- Diagnostics
- Auto-completion
- Hover information

## Verification

1. Open a `.gxc` file in NeoVim
2. Check filetype: `:set filetype?` (should show `gxc`)
3. Check LSP status: `:LspInfo`
4. Test completion: Start typing in a `.gxc` file

## Troubleshooting

**LSP not starting:**
- Verify `galaxy` is in PATH: `which galaxy`
- Check LSP logs: `:LspLog`
- Ensure `galaxy lsp-server` command exists: `galaxy --help | grep lsp`

**No syntax highlighting:**
- Verify filetype is detected: `:set filetype?`
- Check syntax file is loaded: `:scriptnames | grep gxc`

**Galaxy binary not found:**
```bash
# Rebuild and install galaxy
cd /path/to/galaxy
go build -o galaxy ./cmd/galaxy
sudo mv galaxy /usr/local/bin/
```
