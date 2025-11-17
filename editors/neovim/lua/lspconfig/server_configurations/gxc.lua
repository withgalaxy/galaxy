local util = require 'lspconfig.util'

return {
  default_config = {
    cmd = { 'galaxy', 'lsp-server', '--stdio' },
    filetypes = { 'gxc' },
    root_dir = util.root_pattern('galaxy.config.json', 'galaxy.config.toml', '.git'),
    single_file_support = true,
  },
  docs = {
    description = [[
https://github.com/cameron-webmatter/galaxy

Language server for GXC (Galaxy Component) files.
Requires the `galaxy` CLI to be installed and available in PATH.
]],
  },
}
