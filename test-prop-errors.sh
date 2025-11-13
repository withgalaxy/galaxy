#!/bin/bash
# Quick test to verify prop type checking works

cd /Users/cameron/dev/galaxy-mono/galaxy
echo "Running test..."
go test ./pkg/lsp -run "TestTemplateAnalyzer/real_Nav" -v 2>&1 | grep -E "(PASS|FAIL|Type mismatch|diagnostics)"

echo ""
echo "If you see 'PASS' and 'found 1 diagnostics', then prop checking works!"
echo ""
echo "To see it in neovim:"
echo "1. Make sure you've run: go install ./cmd/galaxy"
echo "2. Open nvim and run: :LspRestart"  
echo "3. Open dashboard.gxc with userName={1}"
echo "4. Check for errors with: :lua vim.diagnostic.setqflist()"
