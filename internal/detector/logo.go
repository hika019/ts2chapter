package detector

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"
)

type Frame struct {
	frame  int
	isLogo bool
}

const (
	FrameDir = "frames"

	// MedianWindowSize is the sliding window size for the temporal median filter
	// applied to the logo presence time series.
	MedianWindowSize = 7
)

// GenerateChaptersByLogo detects logo presence using variance-based analysis
// and generates chapters based on logo/CM transitions.
func GenerateChaptersByLogo(tsFile string, mainThreshold float32) error {
	Logger.Println("ロゴ検出によるチャプター生成を開始します...")

	chapters, err := runVarianceLogoPipeline(tsFile)
	if err != nil {
		return err
	}

	return writeChapters(tsFile, chapters)
}

// runVarianceLogoPipeline executes the full variance-based logo detection pipeline.
// Pipeline: grayframes → variance map → logo region → presence → median filter → chapters.
func runVarianceLogoPipeline(tsFile string) ([]Chapter, error) {
	absFramesDir, err := filepath.Abs(FrameDir)
	if err != nil {
		return nil, fmt.Errorf("resolve FrameDir path: %w", err)
	}

	if _, err := os.Stat(absFramesDir); err == nil {
		if err := os.RemoveAll(absFramesDir); err != nil {
			return nil, fmt.Errorf("failed to remove frames dir: %w", err)
		}
	}
	if err := os.MkdirAll(absFramesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create frames dir: %w", err)
	}

	// Step 1: Extract grayscale frames (ffmpeg fps=1,format=gray)
	if err := captureGrayFrames(tsFile, absFramesDir); err != nil {
		return nil, fmt.Errorf("frame extraction failed: %w", err)
	}

	// Step 2: Load frames into memory
	frames, err := loadGrayFrames(absFramesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load frames: %w", err)
	}
	if len(frames) == 0 {
		return nil, fmt.Errorf("no frames extracted")
	}
	Logger.Printf("フレーム数: %d", len(frames))

	// Step 3: Build variance map (Welford's online algorithm)
	vm := BuildVarianceMap(frames)
	Logger.Printf("分散マップ構築完了: %dx%d", vm.Width, vm.Height)

	// Step 4: Detect logo region (OTSU + connected components)
	logoRegion := vm.DetectLogoRegion()
	if logoRegion.Empty() {
		return nil, fmt.Errorf("logo region not detected")
	}
	Logger.Printf("ロゴ領域検出: %v", logoRegion)

	// Step 5: Detect logo presence per frame (MAE vs mean)
	presence := detectLogoPresence(frames, vm, logoRegion)

	// Step 6: Median filter (temporal noise removal)
	filtered := MedianFilterBool(presence, MedianWindowSize)
	Logger.Printf("メディアンフィルタ適用完了 (窓幅=%d)", MedianWindowSize)

	// Step 7: Convert to chapters
	frameInfos := make([]Frame, len(filtered))
	for i, isLogo := range filtered {
		frameInfos[i] = Frame{frame: i, isLogo: isLogo}
	}

	return frameToChapter(frameInfos), nil
}

func frameToChapter(frames []Frame) []Chapter {
	chapters := make([]Chapter, 0)

	for i := 0; i < len(frames)-1; i++ {
		name := ""

		if frames[i].isLogo && !frames[i+1].isLogo {
			name = fmt.Sprintf("CM開始 %d", frames[i].frame)
		} else if !frames[i].isLogo && frames[i+1].isLogo {
			name = fmt.Sprintf("本編 %d", frames[i+1].frame)
		}
		if name != "" {
			chapters = append(chapters, Chapter{
				Time: time.Duration(frames[i].frame) * time.Second,
				Name: name,
			})
		}
	}
	return chapters
}

// captureGrayFrames extracts 1fps grayscale frames from a video using ffmpeg.
func captureGrayFrames(tsFile, framesDir string) error {
	Logger.Println("グレースケールフレーム抽出中...")
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error",
		"-i", tsFile,
		"-vf", "fps=1,format=gray",
		filepath.Join(framesDir, "frame_%06d.jpg"))
	return cmd.Run()
}

// loadGrayFrames loads sorted grayscale JPEG frames from a directory.
func loadGrayFrames(dir string) ([]image.Gray, error) {
	files, err := filepath.Glob(filepath.Join(dir, "frame_*.jpg"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob frames: %w", err)
	}
	sort.Strings(files)

	frames := make([]image.Gray, 0, len(files))
	for _, f := range files {
		img, err := readGrayJPEG(f)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", f, err)
		}
		frames = append(frames, *img)
	}
	Logger.Printf("フレーム読み込み完了: %d 枚", len(frames))
	return frames, nil
}

// detectLogoPresence determines per-frame logo presence by computing MAE
// between each frame's logo region and the mean image.
// Uses OTSU on the MAE distribution for automatic threshold selection.
// Low MAE = similar to mean = logo present; High MAE = logo absent (CM).
func detectLogoPresence(frames []image.Gray, vm *VarianceMap, logoRegion image.Rectangle) []bool {
	if len(frames) == 0 || logoRegion.Empty() {
		return make([]bool, len(frames))
	}

	// Compute MAE for each frame against the mean in the logo region
	maes := make([]float64, len(frames))
	for i := range frames {
		var sum float64
		count := 0
		for y := logoRegion.Min.Y; y < logoRegion.Max.Y; y++ {
			for x := logoRegion.Min.X; x < logoRegion.Max.X; x++ {
				pixVal := float64(frames[i].Pix[y*frames[i].Stride+x])
				meanVal := vm.Mean[y][x]
				diff := pixVal - meanVal
				if diff < 0 {
					diff = -diff
				}
				sum += diff
				count++
			}
		}
		if count > 0 {
			maes[i] = sum / float64(count)
		}
	}

	// OTSU on MAE histogram for automatic threshold
	const maeBins = 256
	maxMAE := 0.0
	for _, m := range maes {
		if m > maxMAE {
			maxMAE = m
		}
	}
	if maxMAE == 0 {
		// All frames identical to mean — treat all as logo present
		result := make([]bool, len(frames))
		for i := range result {
			result[i] = true
		}
		return result
	}

	hist := make([]int, maeBins)
	binWidth := maxMAE / float64(maeBins-1)
	for _, m := range maes {
		bin := int(m / binWidth)
		if bin >= maeBins {
			bin = maeBins - 1
		}
		hist[bin]++
	}

	threshBin := OtsuThreshold(hist)
	maeThreshold := threshBin * binWidth
	Logger.Printf("MAE閾値(OTSU): %.2f", maeThreshold)

	presence := make([]bool, len(frames))
	for i, m := range maes {
		presence[i] = m <= maeThreshold
		DebugLogger.Printf("Frame %d: MAE=%.2f, logo=%v", i, m, presence[i])
	}

	return presence
}

func readGrayJPEG(path string) (*image.Gray, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer f.Close()

	img, err := jpeg.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode %s: %w", path, err)
	}

	if gray, ok := img.(*image.Gray); ok {
		return gray, nil
	}

	bounds := img.Bounds()
	gray := image.NewGray(bounds)
	draw.Draw(gray, bounds, img, bounds.Min, draw.Src)
	return gray, nil
}

func writeGrayJPEG(path string, img *image.Gray) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", path, err)
	}
	defer f.Close()

	return jpeg.Encode(f, img, &jpeg.Options{Quality: 95})
}
