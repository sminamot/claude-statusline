# claude-statusline
![screenshot](assets/screenshot.png)

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
