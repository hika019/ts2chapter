package detector

import (
	"bufio"
	"math"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"time"
)

type SilenceInterval struct {
	Start time.Duration
	End   time.Duration
}

func GenerateChaptersBySilence(tsFile string) error {
	Logger.Println("無音区間を検出中...")
	var chapters []Chapter

	detected, err := detectSilences(tsFile)
	if err != nil {
		return err
	}
	chapters = generateChaptersFromSilences(detected)

	return writeChapters(tsFile, chapters)
}

// ffmpeg を使って無音区間を検出し、start/end ペアとして返す
func detectSilences(tsFile string) ([]SilenceInterval, error) {
	cmd := exec.Command("ffmpeg", "-i", tsFile, "-af", "silencedetect=n=-70dB:d=0.5", "-f", "null", "-")
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(stderr)
	reStart := regexp.MustCompile(`silence_start: (\d+(\.\d+)?)`)
	reEnd := regexp.MustCompile(`silence_end: (\d+(\.\d+)?)`)

	var intervals []SilenceInterval
	var currentStart *time.Duration

	for scanner.Scan() {
		line := scanner.Text()
		if match := reStart.FindStringSubmatch(line); match != nil {
			sec, _ := strconv.ParseFloat(match[1], 64)
			t := time.Duration(sec * float64(time.Second))
			currentStart = &t
		} else if match := reEnd.FindStringSubmatch(line); match != nil {
			if currentStart != nil {
				sec, _ := strconv.ParseFloat(match[1], 64)
				end := time.Duration(sec * float64(time.Second))
				intervals = append(intervals, SilenceInterval{
					Start: *currentStart,
					End:   end,
				})
				currentStart = nil
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	return intervals, nil
}

// 無音区間からチャプターを生成
func generateChaptersFromSilences(silences []SilenceInterval) []Chapter {
	if len(silences) == 0 {
		return []Chapter{{Time: 0, Name: "本編"}}
	}

	// 無音区間の長さでフィルタ (0.3秒〜2秒 = CM境界候補)
	var filtered []SilenceInterval
	for _, s := range silences {
		duration := s.End - s.Start
		if duration >= 300*time.Millisecond && duration <= 2*time.Second {
			filtered = append(filtered, s)
		}
	}

	if len(filtered) == 0 {
		return []Chapter{{Time: 0, Name: "本編"}}
	}

	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Start < filtered[j].Start })

	// CM区間を検出 (連続する無音区間の間隔が15/30/60/90/120秒 ±2秒ならCM)
	cmDurations := []time.Duration{
		15 * time.Second,
		30 * time.Second,
		60 * time.Second,
		90 * time.Second,
		120 * time.Second,
	}
	tolerance := 2 * time.Second

	type segment struct {
		start  time.Duration
		end    time.Duration
		isCM   bool
		isMain bool
	}

	var segments []segment
	currentPos := time.Duration(0)

	for i := 0; i < len(filtered)-1; i++ {
		gapStart := filtered[i].End
		gapEnd := filtered[i+1].Start
		gapDuration := gapEnd - gapStart

		isCM := false
		for _, cmDur := range cmDurations {
			if math.Abs(float64(gapDuration-cmDur)) <= float64(tolerance) {
				isCM = true
				break
			}
		}

		if isCM {
			// CM区間の前に本編があれば追加
			if gapStart > currentPos {
				segments = append(segments, segment{
					start:  currentPos,
					end:    gapStart,
					isMain: true,
				})
			}
			// CM区間を追加
			segments = append(segments, segment{
				start: gapStart,
				end:   gapEnd,
				isCM:  true,
			})
			currentPos = gapEnd
		}
	}

	// 最後の本編区間
	if len(filtered) > 0 {
		lastEnd := filtered[len(filtered)-1].End
		if lastEnd > currentPos {
			segments = append(segments, segment{
				start:  currentPos,
				end:    lastEnd,
				isMain: true,
			})
		}
	}

	// セグメントからチャプターを生成
	var chapters []Chapter
	if len(segments) == 0 {
		return []Chapter{{Time: 0, Name: "本編"}}
	}

	// 最初のセグメントが本編でない場合、冒頭に本編を追加
	if !segments[0].isMain {
		chapters = append(chapters, Chapter{Time: 0, Name: "本編"})
	}

	for _, seg := range segments {
		name := "本編"
		if seg.isCM {
			name = "CM"
		}
		chapters = append(chapters, Chapter{
			Time: seg.start,
			Name: name,
		})
	}

	return chapters
}
