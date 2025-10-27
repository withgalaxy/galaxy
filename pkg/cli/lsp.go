package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/withgalaxy/galaxy/pkg/lsp"
	"github.com/spf13/cobra"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

var lspCmd = &cobra.Command{
	Use:   "lsp-server",
	Short: "Start the GXC language server",
	Long:  `Start the language server protocol (LSP) server for .gxc files`,
	RunE:  runLSP,
}

func init() {
	rootCmd.AddCommand(lspCmd)
	lspCmd.Flags().Bool("stdio", false, "use stdin/stdout for communication")
}

func runLSP(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	stream := jsonrpc2.NewStream(stdrwc{})
	conn := jsonrpc2.NewConn(stream)

	server := lsp.NewServer(conn)

	handler := func(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
		switch req.Method() {
		case "initialize":
			var params protocol.InitializeParams
			if err := json.Unmarshal(req.Params(), &params); err != nil {
				return reply(ctx, nil, err)
			}
			result, err := server.Initialize(ctx, &params)
			return reply(ctx, result, err)

		case "initialized":
			var params protocol.InitializedParams
			if err := json.Unmarshal(req.Params(), &params); err != nil {
				return reply(ctx, nil, err)
			}
			return reply(ctx, nil, server.Initialized(ctx, &params))

		case "textDocument/didOpen":
			var params protocol.DidOpenTextDocumentParams
			if err := json.Unmarshal(req.Params(), &params); err != nil {
				return reply(ctx, nil, err)
			}
			return reply(ctx, nil, server.DidOpen(ctx, &params))

		case "textDocument/didChange":
			var params protocol.DidChangeTextDocumentParams
			if err := json.Unmarshal(req.Params(), &params); err != nil {
				return reply(ctx, nil, err)
			}
			return reply(ctx, nil, server.DidChange(ctx, &params))

		case "textDocument/didClose":
			var params protocol.DidCloseTextDocumentParams
			if err := json.Unmarshal(req.Params(), &params); err != nil {
				return reply(ctx, nil, err)
			}
			return reply(ctx, nil, server.DidClose(ctx, &params))

		case "textDocument/completion":
			var params protocol.CompletionParams
			if err := json.Unmarshal(req.Params(), &params); err != nil {
				return reply(ctx, nil, err)
			}
			result, err := server.Completion(ctx, &params)
			return reply(ctx, result, err)

		case "textDocument/hover":
			var params protocol.HoverParams
			if err := json.Unmarshal(req.Params(), &params); err != nil {
				return reply(ctx, nil, err)
			}
			result, err := server.Hover(ctx, &params)
			return reply(ctx, result, err)

		case "textDocument/definition":
			var params protocol.DefinitionParams
			if err := json.Unmarshal(req.Params(), &params); err != nil {
				return reply(ctx, nil, err)
			}
			result, err := server.Definition(ctx, &params)
			return reply(ctx, result, err)

		case "shutdown":
			return reply(ctx, nil, server.Shutdown(ctx))

		case "exit":
			server.Exit(ctx)
			return reply(ctx, nil, nil)
		}

		return reply(ctx, nil, nil)
	}

	conn.Go(ctx, handler)

	<-conn.Done()

	if err := conn.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Connection error: %v\n", err)
		os.Exit(1)
	}

	return nil
}

type stdrwc struct{}

func (stdrwc) Read(p []byte) (int, error) {
	return os.Stdin.Read(p)
}

func (stdrwc) Write(p []byte) (int, error) {
	return os.Stdout.Write(p)
}

func (stdrwc) Close() error {
	return nil
}
