package detector

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const mergeTolerance = 3 * time.Second

// GenerateChaptersByCombined は無音検出とロゴ検出の結果をマージし、
// 高精度なチャプター生成を行う。
func GenerateChaptersByCombined(tsFile string, mainThreshold float32) error {
	Logger.Println("combined検出によるチャプター生成を開始します...")

	// Step 1: 無音区間を検出
	Logger.Println("Step 1: 無音区間を検出中...")
	silenceTimes, err := detectSilences(tsFile)
	if err != nil {
		return fmt.Errorf("silence detection failed: %w", err)
	}
	silenceChapters := generateChaptersFromSilences(silenceTimes)
	Logger.Printf("無音検出: %d チャプター境界を検出", len(silenceChapters))

	// Step 2: ロゴ検出パイプライン
	Logger.Println("Step 2: ロゴ検出パイプライン開始...")
	logoChapters, err := runLogoPipeline(tsFile, mainThreshold)
	if err != nil {
		return fmt.Errorf("logo detection failed: %w", err)
	}
	Logger.Printf("ロゴ検出: %d チャプター境界を検出", len(logoChapters))

	// Step 3: 両結果をマージ
	Logger.Println("Step 3: 検出結果をマージ中...")
	merged := mergeChapters(silenceChapters, logoChapters)
	Logger.Printf("マージ結果: %d チャプター", len(merged))

	// Step 4: 書き出し
	return writeChapters(tsFile, merged)
}

// runLogoPipeline executes the variance-based logo detection pipeline
// and returns detected chapters.
func runLogoPipeline(tsFile string, mainThreshold float32) ([]Chapter, error) {
	return runVarianceLogoPipeline(tsFile)
}

type mergeCandidate struct {
	time   time.Duration
	name   string
	source string
}

// mergeChapters は無音検出とロゴ検出のチャプター境界をマージする。
// 両方一致する境界は高信頼度として優先採用し、片方のみの境界も補助的に採用する。
// ±3秒以内の境界は同一境界とみなす。
func mergeChapters(silenceChapters, logoChapters []Chapter) []Chapter {
	var candidates []mergeCandidate

	for _, c := range silenceChapters {
		candidates = append(candidates, mergeCandidate{c.Time, c.Name, "silence"})
	}
	for _, c := range logoChapters {
		name := "CM"
		if strings.Contains(c.Name, "本編") {
			name = "本編"
		}
		candidates = append(candidates, mergeCandidate{c.Time, name, "logo"})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].time < candidates[j].time
	})

	var merged []Chapter
	used := make([]bool, len(candidates))

	for i := 0; i < len(candidates); i++ {
		if used[i] {
			continue
		}
		used[i] = true

		bestTime := candidates[i].time
		bestName := candidates[i].name
		matched := false

		// ±3秒以内で異なるソースの境界を探す
		for j := i + 1; j < len(candidates); j++ {
			if used[j] {
				continue
			}
			diff := candidates[j].time - candidates[i].time
			if diff > mergeTolerance {
				break
			}
			if candidates[j].source != candidates[i].source {
				matched = true
				used[j] = true
				// 高信頼度: 両者の平均時刻を採用、名前はロゴ検出を優先
				bestTime = (candidates[i].time + candidates[j].time) / 2
				if candidates[j].source == "logo" {
					bestName = candidates[j].name
				}
				break
			}
		}

		if matched {
			Logger.Printf("高信頼度境界: %s (%s)", formatTime(bestTime), bestName)
		} else {
			Logger.Printf("低信頼度境界: %s (%s, %sのみ)", formatTime(bestTime), bestName, candidates[i].source)
		}

		merged = append(merged, Chapter{Time: bestTime, Name: bestName})
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Time < merged[j].Time
	})

	return merged
}
