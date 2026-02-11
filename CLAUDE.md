# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Claude Code のカスタムステータスライン。stdin から JSON を受け取り、2行のANSI装飾付きステータスラインを stdout に出力する単一バイナリのCLIツール。

## Commands

```bash
# テスト実行
go test ./...

# ビルド & インストール
go install .

# サンプルJSONで動作確認
echo '{"model":{"display_name":"Opus 4.6"},"context_window":{"context_window_size":200000,"current_usage":{"input_tokens":40000,"cache_creation_input_tokens":5000,"cache_read_input_tokens":1000}},"version":"1.0.30","cost":{"total_cost_usd":0.42,"total_duration_ms":900000},"cwd":"/Users/example/project"}' | go run .
```

## Architecture

単一ファイル構成（`main.go`）。外部依存なし（標準ライブラリのみ）。

- `StatusData` struct: stdin JSON のパース用構造体
- `buildProgressBar()`: 8段階Unicode端数ブロック + ANSI背景色によるプログレスバー。色は引数で受け取り、塗り・空きとも同じ背景色（`48;5;236m`）で隙間を解消
- `percentageColor()`: 使用率に応じた4段階ANSIカラー（~50%緑, 50~70%黄, 70~90%オレンジ, 90%~赤）
- `clockEmoji()`: 時刻に応じた30分刻みの時計絵文字（U+1F550〜U+1F567）
- `main()`: stdin→JSON パース→各パーツ組み立て→2行出力

## ANSI Color Conventions

| 項目 | カラー |
|---|---|
| モデル名 | シアン `\x1b[36m` |
| コスト | ゴールド `\x1b[38;2;255;215;0m` |
| ブランチ | パープル `\x1b[35m` |
| 1行目テキスト | ブライトホワイト `\x1b[97m` |
| 2行目(cwd) | グレー `\033[90m` |
| プログレスバー背景 | ダークグレー `\x1b[48;5;236m` |
