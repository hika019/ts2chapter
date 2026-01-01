package detector

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gocv.io/x/gocv"
)

type logoSegment struct {
	Start time.Duration
	End   time.Duration
}

type Frame struct {
	frame  int
	isLogo bool
}

const (
	LogoEdgePath    = "00-logo_edge.jpg"
	LogoMaskPath    = "00-logo_mask.jpg"
	LogoBigMaskPath = "00-logo_big_mask.jpg"
	FrameDir        = "frames"
	EdgesDir        = "edges"
)

// GenerateChaptersByLogo は tsファイルから1秒毎にフレーム抽出し、
// 四隅ロゴの有無でCM区間を検出してチャプターを生成する。
func GenerateChaptersByLogo(tsFile string) error {
	// const FrameDir = "frames"
	// const EdgesDir = "edges"

	if _, err := os.Stat(FrameDir); err != nil {
		err := os.RemoveAll(FrameDir) // 既存のフレームディレクトリを削除
		if err != nil {
			return fmt.Errorf("failed to remove frames dir: %w", err)
		}
	}

	if err := os.MkdirAll(FrameDir, 0755); err != nil {
		return fmt.Errorf("failed to create frames dir: %w", err)
	}

	if err := captureFrames(tsFile, FrameDir); err != nil {
		return err
	}

	if err := edgeMask(FrameDir, EdgesDir); err != nil {
		return err
	}

	err := filepath.WalkDir(EdgesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".jpg") {
			fmt.Println("処理中:", path)
			err := maskImage(path)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to process frames: %w", err)
	}

	edge_ave, err := averageImages(EdgesDir)
	if err != nil {
		return fmt.Errorf("failed to create average image: %w", err)
	}

	ret2 := gocv.Threshold(edge_ave, &edge_ave, 30, 255, gocv.ThresholdBinary)
	fmt.Println("ロゴ推定画像の閾値処理完了")
	fmt.Println("otsu:", ret2)

	gocv.IMWrite(filepath.Join(EdgesDir, LogoEdgePath), edge_ave)
	fmt.Printf("ロゴエッジ画像を書き出しました: %s\n", filepath.Join(EdgesDir, LogoEdgePath))

	// ロゴマスクを生成
	mask := gocv.NewMat()
	err = edgeToMask(edge_ave, &mask, 2, 2)
	if err != nil {
		return err
	}

	if mask.Empty() {
		return err
	}
	gocv.IMWrite(filepath.Join(EdgesDir, LogoMaskPath), mask)
	fmt.Printf("ロゴマスク画像を書き出しました: %s\n", filepath.Join(EdgesDir, LogoMaskPath))

	maskBig := gocv.NewMat()
	err = edgeToMask(edge_ave, &maskBig, 4, 4)
	if err != nil {
		return err
	}
	gocv.IMWrite(filepath.Join(EdgesDir, LogoBigMaskPath), maskBig)
	fmt.Printf("ロゴマスク画像を書き出しました: %s\n", filepath.Join(EdgesDir, LogoBigMaskPath))

	// ロゴマスクの平均明るさを計算
	frames, err := judgeLogo(FrameDir, mask, maskBig, 0.5) // 閾値は0.5
	if err != nil {
		return fmt.Errorf("failed to judge logo presence: %w", err)
	}

	fps, err := getFPS(tsFile)
	if err != nil {
		return fmt.Errorf("failed to get FPS: %w", err)
	}
	fmt.Printf("動画のFPS: %.2f\n", fps)

	chapters := make([]Chapter, 0)

	for i := 0; i < len(frames)-1; i++ {
		if frames[i].isLogo && !frames[i+1].isLogo {
			// ロゴありからロゴなしへ変化した場合、CM区間
			chapters = append(chapters, Chapter{
				Time: time.Duration(frames[i].frame) * time.Second / time.Duration(fps),
				Name: fmt.Sprintf("CM開始 %d", frames[i].frame),
			})
		} else if !frames[i].isLogo && frames[i+1].isLogo {
			// ロゴなしからロゴありへ変化した場合、CM終了
			chapters = append(chapters, Chapter{
				Time: time.Duration(frames[i+1].frame) * time.Second / time.Duration(fps),
				Name: fmt.Sprintf("本編 %d", frames[i+1].frame),
			})
		}
	}

	err = writeChapters(tsFile, chapters)
	if err != nil {
		return fmt.Errorf("failed to write chapters: %w", err)
	}

	return nil
}

func getFPS(tsFile string) (float64, error) {
	cap, err := gocv.VideoCaptureFile(tsFile)
	if err != nil {
		log.Fatalf("動画ファイルを開けません: %v", err)
	}
	defer cap.Close()

	fps := cap.Get(gocv.VideoCaptureFPS)
	return fps, nil
}

func judgeLogo(frameDir string, logoMask gocv.Mat, logoBigMask gocv.Mat, intensityThreshold float64) ([]Frame, error) {

	if logoMask.Empty() || logoBigMask.Empty() {
		return nil, errors.New("ロゴマスクが空です")
	}

	frame := 1
	frameInfos := make([]Frame, 0)

	err := filepath.WalkDir(frameDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".jpg") {
			fmt.Println("処理中:", path)

			img := gocv.IMRead(path, gocv.IMReadGrayScale)
			gocv.ConvertScaleAbs(img, &img, 0.8, 0)
			logoMaskAvg, err := averageBrightnessWithMask(img, logoMask)
			if err != nil {
				return err
			}

			logoBigMaskAve, err := averageBrightnessWithMask(img, logoBigMask)
			if err != nil {
				return err
			}
			img.Close()

			frameInfo := Frame{
				frame:  frame,
				isLogo: logoMaskAvg-logoBigMaskAve > intensityThreshold, // 閾値
			}
			frameInfos = append(frameInfos, frameInfo)
			frame++

			if logoMaskAvg-logoBigMaskAve > intensityThreshold {
				fmt.Println("ロゴあり:", path)
			} else {
				fmt.Println("ロゴなし:", path, logoMaskAvg-logoBigMaskAve)
			}
			// fmt.Printf("ファイル: %s, ロゴマスク平均: %.2f, ロゴビッグマスク平均: %.2f\n", path, logoMaskAvg, logoBigMaskAve)

		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to process frames: %w", err)
	}
	return frameInfos, nil
}

func averageBrightnessWithMask(img gocv.Mat, mask gocv.Mat) (float64, error) {
	if img.Empty() || mask.Empty() {
		return 0, fmt.Errorf("image or mask is empty")
	}
	if img.Rows() != mask.Rows() || img.Cols() != mask.Cols() {
		return 0, fmt.Errorf("image and mask size mismatch")
	}

	var sum float64
	var count int

	rows := img.Rows()
	cols := img.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if mask.GetUCharAt(y, x) == 255 {
				val := img.GetUCharAt(y, x)
				sum += float64(val)
				count++
			}
		}
	}

	if count == 0 {
		return 0, fmt.Errorf("mask has no white pixels")
	}

	return sum / float64(count), nil
}

func edgeToMask(edge gocv.Mat, dest *gocv.Mat, pt_x int, pt_y int) error {
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(pt_x, pt_y))
	gocv.Dilate(edge, &edge, kernel)

	contours := gocv.FindContours(edge, gocv.RetrievalExternal, gocv.ChainApproxSimple)

	// 空のマスク作成（黒）
	mask := gocv.NewMatWithSize(edge.Rows(), edge.Cols(), gocv.MatTypeCV8U)
	defer mask.Close()

	// 輪郭を塗りつぶしで描画
	// 正しい形式 (すでに [][]image.Point)
	err := gocv.FillPoly(&mask, contours, color.RGBA{255, 255, 255, 0})
	if err != nil {
		return fmt.Errorf("failed to fill contours: %w", err)
	}

	*dest = mask.Clone()
	return nil
}

// captureFrames ffmpegで1秒毎にJPEGフレーム抽出
func captureFrames(tsFile, outDir string) error {
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-i", tsFile, "-vf", "fps=1", filepath.Join(outDir, "frame_%06d.jpg"))
	return cmd.Run()
}

// エッジ画像の四隅以外を黒塗りする
func maskImage(filepath string) error {
	img := gocv.IMRead(filepath, gocv.IMReadColor)
	if img.Empty() {
		fmt.Printf("Failed to read image: %s\n", filepath)
		return errors.New(fmt.Sprintf("Failed to read image: %s\n", filepath))
	}
	defer img.Close()

	width := img.Cols()
	height := img.Rows()
	w3 := width / 3
	h5 := height / 5

	black := color.RGBA{0, 0, 0, 255}

	regionsToFill := []image.Rectangle{
		image.Rect(w3, 0, w3*2, h5),      // 上中央
		image.Rect(0, h5, w3, h5*4),      // 左中央
		image.Rect(w3, h5, w3*2, h5*4),   // 中央
		image.Rect(w3*2, h5, w3*3, h5*4), // 右中央
		image.Rect(w3, h5*4, w3*2, h5*5), // 下中央
	}

	for _, r := range regionsToFill {
		gocv.Rectangle(&img, r, black, -1)

	}

	ok := gocv.IMWrite(filepath, img)
	if !ok {
		fmt.Printf("Failed to write image: %s\n", filepath)
		return errors.New(fmt.Sprintf("Failed to write image: %s\n", filepath))
	}
	return nil
}

func edgeMask(frameDir, edgesDir string) error {

	if _, err := os.Stat(edgesDir); err == nil {
		err := os.RemoveAll(edgesDir) // 既存のエッジディレクトリを削除
		if err != nil {
			return fmt.Errorf("failed to remove edges dir: %w", err)
		}
	}

	if err := os.MkdirAll(edgesDir, 0755); err != nil {
		return fmt.Errorf("failed to create edges dir: %w", err)
	}

	filepath.Walk(frameDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// .jpgファイルのみ処理
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".jpg") {
			fmt.Println("Processing:", path)

			img := gocv.IMRead(path, gocv.IMReadColor)
			if img.Empty() {
				fmt.Println("Failed to read:", path)
				return nil
			}
			defer img.Close()

			// グレースケール変換
			gray := gocv.NewMat()
			defer gray.Close()
			gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

			// 明るさ調整
			result := gocv.NewMat()
			defer result.Close()
			gocv.ConvertScaleAbs(gray, &result, 0.8, 0)

			// Cannyエッジ検出
			edges := gocv.NewMat()
			defer edges.Close()
			gocv.Canny(result, &edges, 50, 150)

			// 保存ファイル名を作成
			ext := filepath.Ext(path)
			base := strings.TrimSuffix(filepath.Base(path), ext)
			outPath := filepath.Join(edgesDir, base+"-edge"+ext)

			if ok := gocv.IMWrite(outPath, edges); !ok {
				fmt.Println("Failed to write:", outPath)
			} else {
				fmt.Println("Saved:", outPath)
			}
		}
		return nil
	})
	return nil

}

// averageImages 指定ディレクトリ内のフレーム画像の平均画像を作成する。
// 画像はグレースケールで読み込み、明るさの平均を算出。
func averageImages(dir string) (gocv.Mat, error) {
	fmt.Println("フレーム画像の平均を計算中...")
	files, err := filepath.Glob(filepath.Join(dir, "frame_*.jpg"))
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("failed to list frames: %w", err)
	}
	if len(files) == 0 {
		return gocv.NewMat(), fmt.Errorf("no frame images found")
	}

	base := gocv.IMRead(files[0], gocv.IMReadGrayScale)
	acc := gocv.NewMatWithSize(base.Rows(), base.Cols(), gocv.MatTypeCV64F)
	defer acc.Close()
	fmt.Println("フレーム画像の初期化完了:", base.Size())
	base.Close()

	count := 0
	for _, file := range files {
		fmt.Println("処理中:", file)
		img := gocv.IMRead(file, gocv.IMReadGrayScale)
		floatImg := gocv.NewMat()

		err := img.ConvertTo(&floatImg, gocv.MatTypeCV64F)
		img.Close()
		if err != nil {
			fmt.Printf("画像変換エラー: %s, %v\n", file, err)
			return gocv.NewMat(), err
		}

		err = gocv.Add(acc, floatImg, &acc)
		floatImg.Close()
		if err != nil {
			return gocv.NewMat(), err
		}
		count++
	}

	ave := gocv.NewMatWithSize(acc.Rows(), acc.Cols(), gocv.MatTypeCV8U)
	gocv.ConvertScaleAbs(acc, &ave, 1.0/float64(count), 0)
	fmt.Println(ave.Size(), ave.Type())

	gocv.IMWrite(filepath.Join(dir, "average.jpg"), ave)
	fmt.Printf("ロゴ推定画像を書き出しました: %s\n", filepath.Join(dir, "average.jpg"))

	return ave, nil
}

// detectLogoTimeline フレームごとにロゴマスク領域の明るさを評価し、
// ロゴが消えている区間をロゴ無し（CM区間）として抽出。
func detectLogoTimeline(dir string, logoMask gocv.Mat) ([]logoSegment, error) {
	files, err := filepath.Glob(filepath.Join(dir, "frame_*.jpg"))
	if err != nil {
		return nil, err
	}

	var segments []logoSegment
	var current *logoSegment

	for i, file := range files {
		img := gocv.IMRead(file, gocv.IMReadGrayScale)
		if img.Empty() {
			img.Close()
			return nil, fmt.Errorf("failed to load frame: %s", file)
		}

		score := logoPresenceScore(img, logoMask)
		img.Close()

		t := time.Duration(i) * time.Second

		if score < 0.1 {
			// ロゴなし区間の開始
			if current == nil {
				current = &logoSegment{Start: t}
			}
		} else {
			// ロゴあり区間に戻ったら現在の区間終了
			if current != nil {
				current.End = t
				segments = append(segments, *current)
				current = nil
			}
		}
	}

	// ファイル終端で区間終了していなければ閉じる
	if current != nil {
		current.End = time.Duration(len(files)) * time.Second
		segments = append(segments, *current)
	}

	return segments, nil
}

// logoPresenceScore マスク領域内の明るさ割合を算出（0〜1）
func logoPresenceScore(img, mask gocv.Mat) float64 {
	var overlap, total int

	rows, cols := img.Rows(), img.Cols()

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			if mask.GetUCharAt(y, x) > 0 {
				total++
				if img.GetUCharAt(y, x) > 200 {
					overlap++
				}
			}
		}
	}

	if total == 0 {
		return 1.0
	}
	return float64(overlap) / float64(total)
}

// convertToChapters logoSegment配列をChapter配列に変換
func convertToChapters(segments []logoSegment) []Chapter {
	var chapters []Chapter
	for i, seg := range segments {
		chapters = append(chapters, Chapter{
			Time: seg.Start,
			Name: fmt.Sprintf("CM-%d", i+1),
		})
	}
	return chapters
}
