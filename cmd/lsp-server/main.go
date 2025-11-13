package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/withgalaxy/galaxy/pkg/lsp"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

type readWriteCloser struct {
	*os.File
}

func (rwc readWriteCloser) Read(p []byte) (int, error) {
	return os.Stdin.Read(p)
}

func (rwc readWriteCloser) Write(p []byte) (int, error) {
	return os.Stdout.Write(p)
}

func (rwc readWriteCloser) Close() error {
	return nil
}

func main() {
	log.SetFlags(0)

	ctx := context.Background()

	stream := jsonrpc2.NewStream(readWriteCloser{os.Stdin})
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

		case "textDocument/didSave":
			var params protocol.DidSaveTextDocumentParams
			if err := json.Unmarshal(req.Params(), &params); err != nil {
				return reply(ctx, nil, err)
			}
			return reply(ctx, nil, server.DidSave(ctx, &params))

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
}
