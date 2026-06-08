import * as vscode from 'vscode'
import type { Task } from './types'

const API_BASE = 'http://localhost:8080'
const POLL_INTERVAL_MS = 5000

// ---- Tree Data Provider -----------------------------------------------

class TaskTreeItem extends vscode.TreeItem {
  constructor(public readonly task: Task) {
    super(`${task.target_site}`, vscode.TreeItemCollapsibleState.None)
    this.description = `${task.stage} · ${task.status}`
    this.tooltip = `ID: ${task.id}\nSite: ${task.target_site}\nStatus: ${task.status}`
    this.iconPath = statusIcon(task.status)
    this.command = {
      command: 'testAgents.openReview',
      title: 'Open Review',
      arguments: [task.id],
    }
    this.contextValue = 'testAgentTask'
  }
}

function statusIcon(status: string): vscode.ThemeIcon {
  switch (status) {
    case 'queued':
      return new vscode.ThemeIcon('clock', new vscode.ThemeColor('charts.yellow'))
    case 'running':
      return new vscode.ThemeIcon('sync~spin', new vscode.ThemeColor('charts.blue'))
    case 'big-review-done':
    case 'completed':
      return new vscode.ThemeIcon('check', new vscode.ThemeColor('charts.green'))
    case 'failed':
      return new vscode.ThemeIcon('error', new vscode.ThemeColor('charts.red'))
    default:
      return new vscode.ThemeIcon('circle-outline')
  }
}

class TaskProvider implements vscode.TreeDataProvider<TaskTreeItem> {
  private readonly _onDidChange = new vscode.EventEmitter<TaskTreeItem | undefined | void>()
  readonly onDidChangeTreeData = this._onDidChange.event

  private tasks: Task[] = []

  getTreeItem(element: TaskTreeItem): vscode.TreeItem {
    return element
  }

  getChildren(): TaskTreeItem[] {
    return this.tasks.map((t) => new TaskTreeItem(t))
  }

  async refresh(): Promise<void> {
    try {
      const res = await fetch(`${API_BASE}/api/tasks`)
      if (res.ok) {
        this.tasks = (await res.json()) as Task[]
      }
    } catch {
      // Ingress may not be running; keep existing tasks.
    }
    this._onDidChange.fire()
  }
}

// ---- Webview ----------------------------------------------------------

function openReviewPanel(ctx: vscode.ExtensionContext, taskId: string): void {
  const panel = vscode.window.createWebviewPanel(
    'testAgentsReview',
    `Review: ${taskId.slice(0, 8)}…`,
    vscode.ViewColumn.One,
    { enableScripts: true },
  )

  panel.webview.html = buildWebviewHtml(taskId)
}

function buildWebviewHtml(taskId: string): string {
  const url = `http://localhost:3000/reviews/${taskId}`
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Review ${taskId}</title>
  <style>
    html, body { margin: 0; padding: 0; height: 100vh; background: #1e1e1e; color: #ccc; font-family: var(--vscode-font-family); }
    iframe { width: 100%; height: 100vh; border: 0; }
    .fallback { padding: 24px; }
    a { color: #4fc1ff; }
  </style>
</head>
<body>
  <iframe src="${url}" title="Review panel">
    <div class="fallback">
      <p>Could not embed review panel.</p>
      <p><a href="${url}" target="_blank">Open in browser</a></p>
    </div>
  </iframe>
</body>
</html>`
}

// ---- Polling ----------------------------------------------------------

let notifiedIds = new Set<string>()

async function pollReviewsReady(
  provider: TaskProvider,
): Promise<void> {
  try {
    const res = await fetch(`${API_BASE}/api/reviews/ready`)
    if (!res.ok) return
    const ready = (await res.json()) as Array<{ id: string }>
    for (const r of ready) {
      if (!notifiedIds.has(r.id)) {
        notifiedIds.add(r.id)
        const action = await vscode.window.showInformationMessage(
          `Big review ready for task ${r.id.slice(0, 8)}…`,
          'Open Review',
        )
        if (action === 'Open Review') {
          await vscode.commands.executeCommand('testAgents.openReview', r.id)
        }
      }
    }
  } catch {
    // Ingress offline — silently skip.
  }
  await provider.refresh()
}

// ---- Activation -------------------------------------------------------

export function activate(ctx: vscode.ExtensionContext): void {
  const provider = new TaskProvider()

  ctx.subscriptions.push(
    vscode.window.registerTreeDataProvider('testAgents', provider),
  )

  ctx.subscriptions.push(
    vscode.commands.registerCommand('testAgents.refresh', () => provider.refresh()),
  )

  ctx.subscriptions.push(
    vscode.commands.registerCommand('testAgents.openReview', (id: string) =>
      openReviewPanel(ctx, id),
    ),
  )

  // Initial load.
  void provider.refresh()

  // Poll every 5 s.
  const timer = setInterval(() => void pollReviewsReady(provider), POLL_INTERVAL_MS)
  ctx.subscriptions.push({ dispose: () => clearInterval(timer) })
}

export function deactivate(): void {}
