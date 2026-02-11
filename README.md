# ts2chapter

TS（MPEG-TS）ファイルから自動でチャプター情報を生成するCLIツール。無音区間検出・放送局ロゴ検出・両者の組み合わせによる3つの方式でCM区間を特定し、Matroska形式のチャプターファイルを出力します。

## 必要環境

- Go 1.24.3 以上（ソースからビルドする場合）
- ffmpeg（音声・フレーム解析に使用）
- Docker（コンテナで実行する場合）

## インストール

### ソースからビルド

```bash
git clone https://github.com/hika019/ts2chapter.git
cd ts2chapter
go build -o ts2chapter
```

### Docker イメージのビルド

```bash
docker build -t ts2chapter .
```

## 使い方

### CLI

```bash
# 無音区間によるチャプター生成
ts2chapter silence <TSファイル>

# ロゴ検出によるチャプター生成
ts2chapter logo <TSファイル>

# 無音+ロゴの統合チャプター生成（推奨）
ts2chapter combined <TSファイル>
```

### Docker

```bash
# 入力ファイルのあるディレクトリをマウントして実行
docker run --rm -v /path/to/videos:/data ts2chapter silence /data/recording.ts
docker run --rm -v /path/to/videos:/data ts2chapter combined /data/recording.ts
```

### グローバルオプション

| オプション | 短縮 | 説明 |
|-----------|------|------|
| `--debug` | `-d` | デバッグログを有効化 |

### サブコマンド

#### `silence`

ffmpegの無音検出（silencedetect）を利用してチャプターを生成します。

```bash
ts2chapter silence recording.ts
```

#### `logo`

フレームの分散値解析により放送局ロゴの有無を検出し、本編とCMを区別してチャプターを生成します。

```bash
ts2chapter logo recording.ts
ts2chapter logo --main-threshold 0.6 recording.ts
```

| オプション | デフォルト | 説明 |
|-----------|----------|------|
| `--main-threshold` | `0.55` | 本編検出の一致率しきい値（0.0〜1.0） |

#### `combined`

無音検出とロゴ検出を組み合わせた高精度なチャプター生成を行います。両方の検出結果を統合（mergeTolerance=3秒）して最終的なチャプターを決定します。

```bash
ts2chapter combined recording.ts
ts2chapter combined --main-threshold 0.5 recording.ts
```

| オプション | デフォルト | 説明 |
|-----------|----------|------|
| `--main-threshold` | `0.55` | 本編検出の一致率しきい値（0.0〜1.0） |

## 出力形式

入力ファイルと同じディレクトリに `{ファイル名}.chapter.txt` が生成されます（既存ファイルは上書き）。

フォーマットはMatroska チャプター形式です：

```
CHAPTER01=00:00:00.000
CHAPTER01NAME=Chapter 1
CHAPTER02=00:15:30.500
CHAPTER02NAME=Chapter 2
```

> **注意**: `logo` サブコマンドは処理中に一時的な `frames/` ディレクトリを作成します。

## トラブルシューティング

| 症状 | 原因 | 対処 |
|------|------|------|
| `ffmpeg: command not found` | ffmpegが未インストール | `brew install ffmpeg`（macOS）/ `apt install ffmpeg`（Ubuntu） |
| チャプターが少なすぎる（logo/combined） | しきい値が高すぎる | `--main-threshold` を下げる（例: 0.4） |
| チャプターが多すぎる（logo/combined） | しきい値が低すぎる | `--main-threshold` を上げる（例: 0.7） |
| 出力ファイルが生成されない | 入力ファイルのパスが不正 | TSファイルのパスが正しいか確認 |

## ライセンス

GPL-3.0 — 詳細は [LICENSE](LICENSE) を参照してください。
