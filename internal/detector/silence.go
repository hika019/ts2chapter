package detector

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Chapter struct {
	Time time.Duration
	Name string
}

func GenerateChapters(tsFile string) error {
	var chapters []Chapter

	detected, err := detectSilences(tsFile)
	if err != nil {
		return err
	}
	chapters = generateChaptersFromSilences(detected)

	return writeChapters(tsFile, chapters)
}

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

func writeChapters(tsFile string, chapters []Chapter) error {
	base := strings.TrimSuffix(tsFile, filepath.Ext(tsFile))
	outFile := base + ".chapter.txt"
	f, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer f.Close()

	for i, c := range chapters {
		num := fmt.Sprintf("%02d", i+1)
		fmt.Fprintf(f, "CHAPTER%s=%s\n", num, formatTime(c.Time))
		fmt.Fprintf(f, "CHAPTER%sNAME=%s\n", num, c.Name)
	}

	return nil
}

func formatTime(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	ms := int(d.Milliseconds()) % 1000
	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
}
