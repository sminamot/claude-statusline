package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestFormatTokenCount(t *testing.T) {
	tests := []struct {
		name     string
		tokens   int
		expected string
	}{
		{"zero", 0, "0"},
		{"small number", 500, "500"},
		{"just below 1k", 999, "999"},
		{"exactly 1k", 1000, "1.0k"},
		{"mid thousands", 46000, "46.0k"},
		{"large thousands", 999999, "1000.0k"},
		{"exactly 1M", 1_000_000, "1.0M"},
		{"1.2M", 1_200_000, "1.2M"},
		{"large millions", 5_500_000, "5.5M"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTokenCount(tt.tokens)
			if got != tt.expected {
				t.Errorf("formatTokenCount(%d) = %q, want %q", tt.tokens, got, tt.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		ms       float64
		expected string
	}{
		{"zero", 0, "(0m)"},
		{"1 minute", 60_000, "(1m)"},
		{"15 minutes", 900_000, "(15m)"},
		{"59 minutes", 3_540_000, "(59m)"},
		{"exactly 1 hour", 3_600_000, "(1h0m)"},
		{"1h23m", 4_980_000, "(1h23m)"},
		{"2h30m", 9_000_000, "(2h30m)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.ms)
			if got != tt.expected {
				t.Errorf("formatDuration(%f) = %q, want %q", tt.ms, got, tt.expected)
			}
		})
	}
}

func TestClockEmoji(t *testing.T) {
	tests := []struct {
		hour, minute int
		expected     string
	}{
		{1, 0, "🕐"}, {1, 30, "🕜"},
		{2, 0, "🕑"}, {2, 45, "🕝"},
		{3, 0, "🕒"}, {3, 29, "🕒"},
		{6, 0, "🕕"}, {6, 30, "🕡"},
		{9, 0, "🕘"}, {9, 59, "🕤"},
		{12, 0, "🕛"}, {12, 30, "🕧"},
		{0, 0, "🕛"},   // midnight = 12
		{0, 30, "🕧"},  // midnight:30
		{13, 15, "🕐"}, // PM
		{23, 45, "🕦"}, // 11:45 PM
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d:%02d", tt.hour, tt.minute), func(t *testing.T) {
			got := clockEmoji(tt.hour, tt.minute)
			if got != tt.expected {
				t.Errorf("clockEmoji(%d, %d) = %q, want %q", tt.hour, tt.minute, got, tt.expected)
			}
		})
	}
}

func TestPercentageColor(t *testing.T) {
	tests := []struct {
		name     string
		pct      float64
		expected string
	}{
		{"0% - green", 0, "\x1b[32m"},
		{"25% - green", 25, "\x1b[32m"},
		{"49.9% - green", 49.9, "\x1b[32m"},
		{"50% - yellow", 50, "\x1b[33m"},
		{"60% - yellow", 60, "\x1b[33m"},
		{"69.9% - yellow", 69.9, "\x1b[33m"},
		{"70% - orange", 70, "\x1b[38;5;208m"},
		{"80% - orange", 80, "\x1b[38;5;208m"},
		{"89.9% - orange", 89.9, "\x1b[38;5;208m"},
		{"90% - red", 90, "\x1b[31m"},
		{"95% - red", 95, "\x1b[31m"},
		{"100% - red", 100, "\x1b[31m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := percentageColor(tt.pct)
			if got != tt.expected {
				t.Errorf("percentageColor(%f) = %q, want %q", tt.pct, got, tt.expected)
			}
		})
	}
}

// stripANSI removes ANSI escape sequences for testing bar content
func stripANSI(s string) string {
	var result strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

func TestBuildProgressBar(t *testing.T) {
	const width = 10

	t.Run("0% - all empty", func(t *testing.T) {
		got := stripANSI(buildProgressBar(0, width, ""))
		expected := "          "
		if got != expected {
			t.Errorf("buildProgressBar(0, %d) = %q, want %q", width, got, expected)
		}
	})

	t.Run("100% - all full", func(t *testing.T) {
		got := stripANSI(buildProgressBar(100, width, ""))
		expected := "██████████"
		if got != expected {
			t.Errorf("buildProgressBar(100, %d) = %q, want %q", width, got, expected)
		}
	})

	t.Run("50% - half filled", func(t *testing.T) {
		got := stripANSI(buildProgressBar(50, width, ""))
		fullCount := strings.Count(got, "█")
		spaceCount := strings.Count(got, " ")
		if fullCount != 5 || spaceCount != 5 {
			t.Errorf("buildProgressBar(50, %d) = %q, expected 5 full + 5 space, got %d full + %d space", width, got, fullCount, spaceCount)
		}
	})

	t.Run("width is correct", func(t *testing.T) {
		got := stripANSI(buildProgressBar(33, width, ""))
		runeCount := len([]rune(got))
		if runeCount != width {
			t.Errorf("buildProgressBar(33, %d) rune length = %d, want %d (bar=%q)", width, runeCount, width, got)
		}
	})

	t.Run("partial block for fractional fill", func(t *testing.T) {
		got := stripANSI(buildProgressBar(12.5, width, ""))
		if strings.Count(got, "█") != 1 {
			t.Errorf("buildProgressBar(12.5, %d) = %q, expected 1 full block", width, got)
		}
		runeCount := len([]rune(got))
		if runeCount != width {
			t.Errorf("buildProgressBar(12.5, %d) rune length = %d, want %d", width, runeCount, width)
		}
	})

	t.Run("higher pct has more or equal full blocks", func(t *testing.T) {
		countFull := func(bar string) int {
			return strings.Count(bar, "█")
		}
		for pct := 0.0; pct <= 100; pct += 10 {
			bar := stripANSI(buildProgressBar(pct, width, ""))
			nextBar := stripANSI(buildProgressBar(pct+10, width, ""))
			if pct+10 <= 100 && countFull(nextBar) < countFull(bar) {
				t.Errorf("pct=%.0f has %d full blocks, but pct=%.0f has %d full blocks",
					pct, countFull(bar), pct+10, countFull(nextBar))
			}
		}
	})

	t.Run("all bars have correct width", func(t *testing.T) {
		for pct := 0.0; pct <= 100; pct += 5 {
			bar := stripANSI(buildProgressBar(pct, width, ""))
			runeCount := len([]rune(bar))
			if runeCount != width {
				t.Errorf("buildProgressBar(%.0f, %d) rune length = %d, want %d", pct, width, runeCount, width)
			}
		}
	})
}
