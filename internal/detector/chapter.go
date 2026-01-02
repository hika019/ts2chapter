package detector

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Chapter struct {
	Time time.Duration
	Name string
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
