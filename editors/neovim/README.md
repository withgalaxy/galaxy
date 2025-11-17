# GXC Language Support for Neovim

Language server support for `.gxc` files (Galaxy components) and `galaxy.config.toml` in Neovim.

## Features

- **Syntax Highlighting**: Frontmatter (Go), HTML, CSS, JavaScript, WASM DOM API
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

## Quick Start

### Option 1: Copy Everything (Recommended)

Copy all Galaxy nvim files to your config:

```bash
cp -r editors/neovim/ftdetect ~/.config/nvim/
cp -r editors/neovim/syntax ~/.config/nvim/
cp -r editors/neovim/lua ~/.config/nvim/
```

Then add to your LSP configuration (e.g., `~/.config/nvim/after/plugin/lsp.lua`):

```lua
local lspconfig = require('lspconfig')
local lsp_configurations = require('lspconfig.configs')

-- Galaxy .gxc LSP configuration
if not lsp_configurations.gxc then
  lsp_configurations.gxc = {
    default_config = {
      cmd = { 'galaxy', 'lsp-server', '--stdio' },
      filetypes = { 'gxc' },
      root_dir = lspconfig.util.root_pattern('galaxy.config.toml', 'galaxy.config.json', '.git'),
      single_file_support = true,
      settings = {},
    },
  }
end

lspconfig.gxc.setup({
  on_attach = function(client, bufnr)
    -- Your standard LSP keybindings here
  end,
})
```

### Option 2: Manual Setup

#### 1. File Type Detection

Create `~/.config/nvim/ftdetect/gxc.vim`:

```vim
au BufRead,BufNewFile *.gxc setfiletype gxc
```

#### 2. LSP Configuration

Add to your LSP setup:

```lua
local lspconfig = require('lspconfig')
local configs = require('lspconfig.configs')

if not configs.gxc then
  configs.gxc = {
    default_config = {
      cmd = { 'galaxy', 'lsp-server', '--stdio' },
      filetypes = { 'gxc' },
      root_dir = lspconfig.util.root_pattern('galaxy.config.toml', 'galaxy.config.json', '.git'),
      single_file_support = true,
      settings = {},
    },
  }
end

lspconfig.gxc.setup({})
```

#### 3. Syntax Highlighting

Create `~/.config/nvim/syntax/gxc.vim`:

```vim
if exists("b:current_syntax")
  finish
endif

let b:current_syntax = ""
unlet b:current_syntax
runtime! syntax/html.vim
unlet b:current_syntax

let b:current_syntax = ""
unlet b:current_syntax
syn include @CSS syntax/css.vim
unlet b:current_syntax

let b:current_syntax = ""
unlet b:current_syntax
syn include @Go syntax/go.vim
unlet b:current_syntax

" Frontmatter with full Go support
syn region gxcFrontmatter start=/\%^---$/ end=/^---$/ contains=@Go,gxcGalaxyAPI,gxcGalaxyMethods

" Galaxy-specific API
syn keyword gxcGalaxyAPI Galaxy contained
syn match gxcGalaxyMethods /Galaxy\.\(Redirect\|Locals\|Params\)/ contained

" WASM DOM API
syn keyword gxcWasmDOM wasmdom GetElementById QuerySelector QuerySelectorAll CreateElement contained
syn keyword gxcWasmDOM AddEventListener SetTextContent SetInnerHTML GetInnerHTML contained
syn keyword gxcWasmDOM Fetch FetchWithOptions Alert ConsoleLog ConsoleError contained

" Script regions
syn region gxcScriptGo matchgroup=htmlTag start=+<script[^>]*>+ end=+</script>+ contains=@Go,gxcWasmDOM,gxcGalaxyAPI keepend

" Style region
syn region gxcStyle matchgroup=htmlTag start=+<style[^>]*>+ end=+</style>+ contains=@CSS keepend

" Interpolation in template
syn region gxcInterpolation start=/{/ end=/}/ contained containedin=htmlString,htmlValue,htmlTag

" Galaxy directives
syn match gxcDirective /galaxy:\(if\|for\|else\|elsif\)/ contained containedin=htmlTag

" Highlighting
hi def link gxcGalaxyAPI Special
hi def link gxcGalaxyMethods Function
hi def link gxcWasmDOM Function
hi def link gxcDirective Special
hi def link gxcInterpolation Identifier

let b:current_syntax = "gxc"
```

## Advanced Configuration

### File Type Settings

Create `~/.config/nvim/ftplugin/gxc.lua` for GXC-specific settings:

```lua
-- Comments: default to Go-style for frontmatter
vim.bo.commentstring = '// %s'

-- Indentation
vim.bo.tabstop = 2
vim.bo.shiftwidth = 2
vim.bo.expandtab = true
vim.bo.smartindent = true

-- Folding
vim.wo.foldmethod = 'syntax'
vim.wo.foldlevel = 99

-- File-specific settings
vim.bo.fileencoding = 'utf-8'
```

### Custom Keybindings and Highlighting

Create `~/.config/nvim/after/plugin/gxc.lua`:

```lua
-- Custom highlighting for Galaxy-specific syntax
vim.api.nvim_create_autocmd('FileType', {
  pattern = 'gxc',
  callback = function()
    vim.cmd([[
      highlight link gxcGalaxyAPI Special
      highlight link gxcGalaxyMethods Function
      highlight link gxcWasmDOM Function
      highlight link gxcGoBuildTag PreProc
      highlight link gxcDirective Special
      highlight link gxcInterpolation Identifier
    ]])
  end,
})

-- Keybindings for .gxc navigation
vim.api.nvim_create_autocmd('FileType', {
  pattern = 'gxc',
  callback = function(args)
    local bufnr = args.buf
    
    -- Jump to frontmatter
    vim.keymap.set('n', '<leader>gf', '/^---$<CR>:noh<CR>', 
      { buffer = bufnr, desc = '[G]alaxy jump to [F]rontmatter', silent = true })
    
    -- Jump to template (after frontmatter)
    vim.keymap.set('n', '<leader>gt', '/^---$<CR>n:noh<CR>', 
      { buffer = bufnr, desc = '[G]alaxy jump to [T]emplate', silent = true })
    
    -- Jump to script
    vim.keymap.set('n', '<leader>gs', '/<script<CR>:noh<CR>', 
      { buffer = bufnr, desc = '[G]alaxy jump to [S]cript', silent = true })
    
    -- Jump to style
    vim.keymap.set('n', '<leader>gy', '/<style<CR>:noh<CR>', 
      { buffer = bufnr, desc = '[G]alaxy jump to st[Y]le', silent = true })
  end,
})
```

### Using with lsp-zero

If you use `lsp-zero`, integrate like this:

```lua
local lsp = require("lsp-zero")

lsp.preset("recommended")

-- Your other LSP setup...

local lspconfig = require('lspconfig')
local lsp_configurations = require('lspconfig.configs')

-- Galaxy .gxc LSP configuration
if not lsp_configurations.gxc then
  lsp_configurations.gxc = {
    default_config = {
      cmd = { 'galaxy', 'lsp-server', '--stdio' },
      filetypes = { 'gxc' },
      root_dir = lspconfig.util.root_pattern('galaxy.config.toml', 'galaxy.config.json', '.git'),
      single_file_support = true,
      settings = {},
    },
  }
end

lspconfig.gxc.setup({
  on_attach = function(client, bufnr)
    -- Standard LSP keybindings from lsp-zero
    lsp.on_attach(client, bufnr)
  end,
})

lsp.setup()

-- Ensure gd works for gxc files
vim.api.nvim_create_autocmd("FileType", {
  pattern = "gxc",
  callback = function()
    vim.keymap.set("n", "gd", vim.lsp.buf.definition, { buffer = true, remap = false })
  end,
})
```

### Using lazy.nvim

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
            cmd = { "galaxy", "lsp-server", "--stdio" },
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
          cmd = { "galaxy", "lsp-server", "--stdio" },
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
      cmd = { "galaxy", "lsp-server", "--stdio" },
      root_dir = vim.fs.dirname(vim.fs.find({"galaxy.config.toml"}, { upward = true })[1]),
    })
  end,
})
```

## Syntax Highlighting Features

The included syntax file provides:

- **Frontmatter**: Full Go syntax highlighting between `---` markers
- **Galaxy API**: Highlighting for `Galaxy.Redirect()`, `Galaxy.Locals`, `Galaxy.Params`
- **WASM DOM API**: Highlighting for wasmdom functions like `GetElementById`, `Fetch`, etc.
- **Directives**: `galaxy:if`, `galaxy:for`, `galaxy:else`, `galaxy:elsif`
- **Interpolation**: Variables in `{curly braces}` throughout the template
- **Script tags**: Go/WASM code in `<script>` tags
- **Style tags**: CSS in `<style>` tags
- **HTML**: Full HTML syntax support in template section

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
    cmd = { "/full/path/to/galaxy", "lsp-server", "--stdio" },
    -- ... rest of config
  },
}
```

### No syntax highlighting

1. Verify filetype is detected:
```vim
:set filetype?
```

2. Check if syntax file is loaded:
```vim
:scriptnames | grep gxc
```

3. Reload syntax manually:
```vim
:set syntax=gxc
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

<script>
// WASM code
import "github.com/withgalaxy/galaxy/pkg/wasmdom"

func handleClick() {
  elem := wasmdom.GetElementById("myButton")
  wasmdom.SetTextContent(elem, "Clicked!")
}
</script>
```

## Additional Resources

- [Galaxy Documentation](https://github.com/withgalaxy/galaxy)
- [nvim-lspconfig](https://github.com/neovim/nvim-lspconfig)
- [LSP Configuration Guide](https://github.com/neovim/nvim-lspconfig/blob/master/doc/server_configurations.md)
