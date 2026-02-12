# claude-statusline
<img width="702" height="100" alt="image" src="https://github.com/user-attachments/assets/11a03e43-6aab-487d-aaca-049d4eca69f3" />
<img width="702" height="100" alt="image" src="https://github.com/user-attachments/assets/3616a60b-24d5-4583-83c9-c9882413b1a4" />
<img width="702" height="100" alt="image" src="https://github.com/user-attachments/assets/042a1acc-c93c-4a3b-8ad7-b54f26be5e3e" />
<img width="702" height="100" alt="image" src="https://github.com/user-attachments/assets/2e308064-12d3-418a-88ab-8eae3a8f984d" />

## Install

```bash
go install github.com/sminamot/claude-statusline@latest
```

## Setting

`~/.claude/settings.json`

```json
{
  "statusLine": {
    "type": "command",
    "command": "CLAUDE_STATUSLINE_CONTEXT_LIMIT_PCT=83.5 claude-statusline"
  }
}
```

### CLAUDE_STATUSLINE_CONTEXT_LIMIT_PCT

Specifies the percentage of the context window at which compaction occurs. Defaults to `100`.

For example, setting it to `83.5` treats 83.5% of `context_window_size` as 100% on the progress bar, so the bar fills up right when compaction triggers.

You can check the Autocompact buffer value by running the `/context` command in Claude Code.
