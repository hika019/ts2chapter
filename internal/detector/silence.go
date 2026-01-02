package detector

import (
	"bufio"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"time"
)

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

// ffmpeg を使って無音区間を検出
func detectSilences(tsFile string) ([]time.Duration, error) {
	cmd := exec.Command("ffmpeg", "-i", tsFile, "-af", "silencedetect=n=-70dB:d=0.5", "-f", "null", "-")
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(stderr)
	var times []time.Duration
	re := regexp.MustCompile(`silence_(start|end): (\d+(\.\d+)?)`)

	for scanner.Scan() {
		line := scanner.Text()
		if match := re.FindStringSubmatch(line); match != nil {
			sec, _ := strconv.ParseFloat(match[2], 64)
			times = append(times, time.Duration(sec*float64(time.Second)))
		}
	}

	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	return times, nil
}

// 無音区間からチャプターを生成
func generateChaptersFromSilences(silences []time.Duration) []Chapter {
	sort.Slice(silences, func(i, j int) bool { return silences[i] < silences[j] })
	if len(silences) == 0 {
		return []Chapter{{Time: 0, Name: "本編"}}
	}
	// silences: start,end,start,end,...
	var merged []time.Duration
	const mergeGap = time.Second // 1秒以内を連続と見なす
	var prev time.Duration

	for _, t := range silences {
		if len(merged)%2 == 1 {
			// end time vs next start
			if t-prev <= mergeGap {
				// skip standalone boundary
				continue
			}
		}
		merged = append(merged, t)
		prev = t
	}
	// チャプター生成
	var chapters []Chapter
	for i := 0; i < len(merged); i++ {
		name := "本編"
		if i%2 == 0 {
			name = "CM"
		}
		chapters = append(chapters, Chapter{
			Time: merged[i],
			Name: name,
		})
	}
	return chapters
}
