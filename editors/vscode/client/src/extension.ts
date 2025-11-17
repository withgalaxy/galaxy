import * as path from 'path';
import * as vscode from 'vscode';
import * as fs from 'fs';
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind
} from 'vscode-languageclient/node';

let client: LanguageClient;

function findWorkspaceRoot(uri: vscode.Uri): string | undefined {
  let dir = path.dirname(uri.fsPath);
  const patterns = ['galaxy.config.json', 'galaxy.config.toml', '.git'];
  
  while (dir !== path.dirname(dir)) {
    for (const pattern of patterns) {
      if (fs.existsSync(path.join(dir, pattern))) {
        return dir;
      }
    }
    dir = path.dirname(dir);
  }
  
  return vscode.workspace.getWorkspaceFolder(uri)?.uri.fsPath;
}

export function activate(context: vscode.ExtensionContext) {
  const config = vscode.workspace.getConfiguration('gxc');
  
  if (!config.get('lsp.enable')) {
    return;
  }

  const serverPath = config.get<string>('lsp.serverPath') || 'galaxy';
  
  const serverOptions: ServerOptions = {
    command: serverPath,
    args: ['lsp-server', '--stdio'],
    transport: TransportKind.stdio
  };

  const clientOptions: LanguageClientOptions = {
    documentSelector: [
      { scheme: 'file', language: 'gxc' },
      { scheme: 'file', pattern: '**/galaxy.config.toml' }
    ],
    synchronize: {
      fileEvents: vscode.workspace.createFileSystemWatcher('**/*.{gxc,toml}')
    },
    workspaceFolder: vscode.workspace.workspaceFolders?.[0],
    initializationOptions: {}
  };

  client = new LanguageClient(
    'gxcLanguageServer',
    'GXC Language Server',
    serverOptions,
    clientOptions
  );

  client.start();
}

export function deactivate(): Thenable<void> | undefined {
  if (!client) {
    return undefined;
  }
  return client.stop();
}
