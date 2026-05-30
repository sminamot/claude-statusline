package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type StatusData struct {
	Model struct {
		DisplayName string `json:"display_name"`
	} `json:"model"`
	ContextWindow struct {
		ContextWindowSize int `json:"context_window_size"`
		CurrentUsage      struct {
			InputTokens              int `json:"input_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		} `json:"current_usage"`
	} `json:"context_window"`
	Version string `json:"version"`
	Cost    struct {
		TotalCostUSD    float64 `json:"total_cost_usd"`
		TotalDurationMs float64 `json:"total_duration_ms"`
	} `json:"cost"`
	Cwd        string `json:"cwd"`
	RateLimits *struct {
		FiveHour *RateWindow `json:"five_hour"`
		SevenDay *RateWindow `json:"seven_day"`
	} `json:"rate_limits"`
}

// RateWindow は rate_limits.five_hour / seven_day の各ウィンドウ。
// Claude.ai サブスクライバ(Pro/Max)のみ、かつセッション最初のAPIレスポンス後に出現し、
// 各ウィンドウは独立して欠落しうるためポインタで保持して欠落と0%を区別する。
type RateWindow struct {
	UsedPercentage float64 `json:"used_percentage"`
	ResetsAt       int64   `json:"resets_at"`
}

func formatTokenCount(tokens int) string {
	if tokens >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(tokens)/1_000_000)
	}
	if tokens >= 1_000 {
		return fmt.Sprintf("%.1fk", float64(tokens)/1_000)
	}
	return fmt.Sprintf("%d", tokens)
}

func clockEmoji(hour, minute int) string {
	// 🕐(1:00)=U+1F550, 🕑(2:00)=U+1F551, ... 🕛(12:00)=U+1F55B
	// 🕜(1:30)=U+1F55C, 🕝(2:30)=U+1F55D, ... 🕧(12:30)=U+1F567
	h := hour % 12
	if h == 0 {
		h = 12
	}
	idx := h - 1 // 0-11
	if minute >= 30 {
		return string(rune(0x1F55C + idx))
	}
	return string(rune(0x1F550 + idx))
}

func formatDuration(ms float64) string {
	totalMinutes := int(ms / 60_000)
	if totalMinutes < 60 {
		return fmt.Sprintf("(%dm)", totalMinutes)
	}
	hours := totalMinutes / 60
	minutes := totalMinutes % 60
	return fmt.Sprintf("(%dh%dm)", hours, minutes)
}

func percentageColor(pct float64) string {
	switch {
	case pct >= 90:
		return "\x1b[31m"
	case pct >= 70:
		return "\x1b[38;5;208m"
	case pct >= 50:
		return "\x1b[33m"
	default:
		return "\x1b[32m"
	}
}

func buildProgressBar(pct float64, width int, color string) string {
	const bgDark = "\x1b[48;5;236m" // 暗めグレー背景
	const reset = "\x1b[0m"
	blocks := []rune{'▏', '▎', '▍', '▌', '▋', '▊', '▉'}
	totalSteps := pct / 100 * float64(width) * 8
	fullBlocks := int(totalSteps) / 8
	remainder := int(totalSteps) % 8

	var sb strings.Builder
	// 塗り部分: 前景色 + 背景色
	sb.WriteString(color + bgDark)
	for i := 0; i < fullBlocks && i < width; i++ {
		sb.WriteRune('█')
	}
	if remainder > 0 && fullBlocks < width {
		sb.WriteRune(blocks[remainder-1])
		fullBlocks++
	}
	// 空き部分: 背景色のみでスペース
	for i := fullBlocks; i < width; i++ {
		sb.WriteRune(' ')
	}
	sb.WriteString(reset)
	return sb.String()
}

func getGitBranch(dir string) string {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// rateBarWidth はレート制限プログレスバーの幅（既存contextバーと統一）
const rateBarWidth = 7

// formatResetTime はリセット時刻(Unix epoch秒)を絶対時刻文字列に整形する。
// 同日なら "15:04"、別日なら "1/2 15:04"。resets_at が欠落(0)なら空文字。
// time.Now()依存を避けてテスト可能にするため now を引数で受け取る。
func formatResetTime(resetsAt int64, now time.Time) string {
	if resetsAt == 0 {
		return ""
	}
	t := time.Unix(resetsAt, 0)
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return t.Format("15:04")
	}
	return t.Format("1/2 15:04")
}

// renderRateWindow は1つのレート制限ウィンドウ(5h/7d)の表示セグメントを組み立てる。
// w が nil(欠落)なら空文字を返し、呼び出し側で連結時にスキップする。
func renderRateWindow(label string, w *RateWindow, now time.Time) string {
	if w == nil {
		return ""
	}
	pctDisplay := math.Floor(w.UsedPercentage*10) / 10
	color := percentageColor(pctDisplay)
	bar := buildProgressBar(w.UsedPercentage, rateBarWidth, color)
	seg := fmt.Sprintf("\x1b[97m%s\x1b[0m %s %s%.1f%%\x1b[0m", label, bar, color, pctDisplay)
	if reset := formatResetTime(w.ResetsAt, now); reset != "" {
		seg += fmt.Sprintf(" \x1b[90m(~%s)\x1b[0m", reset)
	}
	return seg
}

func main() {
	var data StatusData
	if err := json.NewDecoder(os.Stdin).Decode(&data); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse JSON: %v\n", err)
		os.Exit(1)
	}

	usage := data.ContextWindow.CurrentUsage
	totalTokens := usage.InputTokens + usage.CacheCreationInputTokens + usage.CacheReadInputTokens

	// CLAUDE_STATUSLINE_CONTEXT_LIMIT_PCT: compaction発生点のパーセンテージ（デフォルト80）
	// context_window_size * limitPct% を100%として表示する
	limitPct := 100.0
	if v := os.Getenv("CLAUDE_STATUSLINE_CONTEXT_LIMIT_PCT"); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed > 0 && parsed <= 100 {
			limitPct = parsed
		}
	}
	effectiveMax := float64(data.ContextWindow.ContextWindowSize) * limitPct / 100
	var pct float64
	if effectiveMax > 0 {
		pct = float64(totalTokens) / effectiveMax * 100
	}

	now := time.Now()
	timeStr := now.Format("15:04")

	var parts []string

	// モデル名（ターコイズ）
	parts = append(parts, fmt.Sprintf("\x1b[36m%s\x1b[0m", data.Model.DisplayName))

	// パーセンテージを小数1桁で切り捨て（表示と色判定を一致させる）
	pctDisplay := math.Floor(pct*10) / 10
	colorCode := percentageColor(pctDisplay)
	bar := buildProgressBar(pct, 10, colorCode)
	tokenStr := formatTokenCount(totalTokens)
	contextPart := fmt.Sprintf("%s\x1b[97m %s %s(%.1f%%)\x1b[0m", bar, tokenStr, colorCode, pctDisplay)
	parts = append(parts, contextPart)

	// バージョン
	parts = append(parts, "v"+data.Version)

	// コスト（ゴールド #FFD700）
	parts = append(parts, fmt.Sprintf("\x1b[38;2;255;215;0m💰$%.2f\x1b[0m", data.Cost.TotalCostUSD))

	// ブランチ（パープル、取得失敗時は省略）
	branch := getGitBranch(data.Cwd)
	if branch != "" {
		parts = append(parts, "\x1b[35m⎇ "+branch+"\x1b[0m")
	}

	// 時刻 + 経過時間
	clock := clockEmoji(now.Hour(), now.Minute())
	duration := formatDuration(data.Cost.TotalDurationMs)
	parts = append(parts, fmt.Sprintf("%s %s %s", clock, timeStr, duration))

	line1 := "\x1b[97m" + strings.Join(parts, " \x1b[0m│\x1b[97m ") + "\x1b[0m"

	// 2行目: cwd（グレー表示、~置換）
	cwd := data.Cwd
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(cwd, home) {
		cwd = "~" + cwd[len(home):]
	}
	line2 := fmt.Sprintf("\x1b[90m%s\x1b[0m", cwd)

	fmt.Println(line1)
	fmt.Println(line2)

	// 3行目: プラン使用制限（5時間/週間）。rate_limits や各ウィンドウは
	// 欠落しうる（Pro/Max以外・セッション開始直後など）ので、存在するものだけ表示。
	if data.RateLimits != nil {
		var segs []string
		if s := renderRateWindow("5h", data.RateLimits.FiveHour, now); s != "" {
			segs = append(segs, s)
		}
		if s := renderRateWindow("7d", data.RateLimits.SevenDay, now); s != "" {
			segs = append(segs, s)
		}
		if len(segs) > 0 {
			fmt.Println("⏳ " + strings.Join(segs, "  "))
		}
	}
}
