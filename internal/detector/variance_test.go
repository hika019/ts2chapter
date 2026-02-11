package detector

import (
	"image"
	"math"
	"testing"
)

func TestBuildVarianceMap_Empty(t *testing.T) {
	vm := BuildVarianceMap(nil)
	if vm.Width != 0 || vm.Height != 0 {
		t.Errorf("Expected empty VarianceMap, got %dx%d", vm.Width, vm.Height)
	}
}

func TestBuildVarianceMap_Constant(t *testing.T) {
	frame := image.NewGray(image.Rect(0, 0, 4, 4))
	for i := range frame.Pix {
		frame.Pix[i] = 100
	}
	frames := []image.Gray{*frame, *frame, *frame}
	vm := BuildVarianceMap(frames)

	if vm.Width != 4 || vm.Height != 4 {
		t.Fatalf("Size = %dx%d, want 4x4", vm.Width, vm.Height)
	}
	for y := 0; y < vm.Height; y++ {
		for x := 0; x < vm.Width; x++ {
			if vm.Mean[y][x] != 100 {
				t.Errorf("Mean[%d][%d] = %f, want 100", y, x, vm.Mean[y][x])
			}
			if vm.Variance[y][x] != 0 {
				t.Errorf("Variance[%d][%d] = %f, want 0", y, x, vm.Variance[y][x])
			}
		}
	}
}

func TestBuildVarianceMap_KnownValues(t *testing.T) {
	// 3 frames: all pixels 10, 20, 30.
	// Mean = 20, Variance = ((10-20)^2 + (20-20)^2 + (30-20)^2) / 3 = 200/3
	frames := make([]image.Gray, 3)
	for i := 0; i < 3; i++ {
		g := image.NewGray(image.Rect(0, 0, 2, 2))
		val := uint8(10 + i*10)
		for j := range g.Pix {
			g.Pix[j] = val
		}
		frames[i] = *g
	}
	vm := BuildVarianceMap(frames)

	wantMean := 20.0
	wantVar := 200.0 / 3.0
	for y := 0; y < vm.Height; y++ {
		for x := 0; x < vm.Width; x++ {
			if math.Abs(vm.Mean[y][x]-wantMean) > 1e-9 {
				t.Errorf("Mean[%d][%d] = %f, want %f", y, x, vm.Mean[y][x], wantMean)
			}
			if math.Abs(vm.Variance[y][x]-wantVar) > 1e-9 {
				t.Errorf("Variance[%d][%d] = %f, want %f", y, x, vm.Variance[y][x], wantVar)
			}
		}
	}
}

func TestBuildVarianceMap_SingleFrame(t *testing.T) {
	frame := image.NewGray(image.Rect(0, 0, 3, 3))
	for i := range frame.Pix {
		frame.Pix[i] = uint8(i * 10)
	}
	vm := BuildVarianceMap([]image.Gray{*frame})

	// Single frame: variance must be 0, mean = pixel value.
	for y := 0; y < vm.Height; y++ {
		for x := 0; x < vm.Width; x++ {
			wantMean := float64(frame.Pix[y*frame.Stride+x])
			if vm.Mean[y][x] != wantMean {
				t.Errorf("Mean[%d][%d] = %f, want %f", y, x, vm.Mean[y][x], wantMean)
			}
			if vm.Variance[y][x] != 0 {
				t.Errorf("Variance[%d][%d] = %f, want 0", y, x, vm.Variance[y][x])
			}
		}
	}
}

func TestOtsuThreshold_Bimodal(t *testing.T) {
	hist := make([]int, 256)
	hist[10] = 1000
	hist[200] = 1000
	threshold := OtsuThreshold(hist)
	// With two perfect spikes, any t in [10,199] gives max sigma.
	// The algorithm returns the first: 10.
	if threshold != 10 {
		t.Errorf("OtsuThreshold = %f, want 10", threshold)
	}
}

func TestOtsuThreshold_Valley(t *testing.T) {
	hist := make([]int, 256)
	for i := 30; i <= 70; i++ {
		hist[i] = 100
	}
	for i := 180; i <= 220; i++ {
		hist[i] = 100
	}
	threshold := OtsuThreshold(hist)
	if threshold < 30 || threshold > 220 {
		t.Errorf("OtsuThreshold = %f, want between 30 and 220", threshold)
	}
}

func TestOtsuThreshold_Empty(t *testing.T) {
	threshold := OtsuThreshold(make([]int, 256))
	if threshold != 0 {
		t.Errorf("OtsuThreshold = %f, want 0", threshold)
	}
}

func TestDetectLogoRegion_WithLogo(t *testing.T) {
	w, h := 100, 100
	vm := &VarianceMap{
		Width:    w,
		Height:   h,
		Mean:     make([][]float64, h),
		Variance: make([][]float64, h),
	}
	for y := 0; y < h; y++ {
		vm.Mean[y] = make([]float64, w)
		vm.Variance[y] = make([]float64, w)
		for x := 0; x < w; x++ {
			vm.Variance[y][x] = 1000 // high variance background
		}
	}
	// Low-variance logo region at (10,10)-(30,30).
	for y := 10; y < 30; y++ {
		for x := 10; x < 30; x++ {
			vm.Variance[y][x] = 1
		}
	}

	rect := vm.DetectLogoRegion()
	want := image.Rect(10, 10, 30, 30)
	if rect != want {
		t.Errorf("DetectLogoRegion = %v, want %v", rect, want)
	}
}

func TestDetectLogoRegion_AllZeroVariance(t *testing.T) {
	w, h := 10, 10
	vm := &VarianceMap{
		Width:    w,
		Height:   h,
		Mean:     make([][]float64, h),
		Variance: make([][]float64, h),
	}
	for y := 0; y < h; y++ {
		vm.Mean[y] = make([]float64, w)
		vm.Variance[y] = make([]float64, w)
	}

	rect := vm.DetectLogoRegion()
	if !rect.Empty() {
		t.Errorf("DetectLogoRegion = %v, want empty", rect)
	}
}

func TestDetectLogoRegion_EmptyMap(t *testing.T) {
	vm := &VarianceMap{}
	rect := vm.DetectLogoRegion()
	if !rect.Empty() {
		t.Errorf("DetectLogoRegion = %v, want empty", rect)
	}
}

func TestLargestComponentBounds_TwoRegions(t *testing.T) {
	w, h := 6, 6
	binary := make([]bool, h*w)
	// Small region: (0,0)-(1,1) = 4 pixels
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			binary[y*w+x] = true
		}
	}
	// Large region: (3,3)-(5,5) = 9 pixels
	for y := 3; y < 6; y++ {
		for x := 3; x < 6; x++ {
			binary[y*w+x] = true
		}
	}

	rect := largestComponentBounds(binary, w, h)
	want := image.Rect(3, 3, 6, 6)
	if rect != want {
		t.Errorf("largestComponentBounds = %v, want %v", rect, want)
	}
}

func TestLargestComponentBounds_LShape(t *testing.T) {
	// L-shaped region should be one connected component.
	w, h := 5, 5
	binary := make([]bool, h*w)
	// Vertical bar: x=0, y=0..3
	for y := 0; y < 4; y++ {
		binary[y*w+0] = true
	}
	// Horizontal bar: y=3, x=0..3
	for x := 0; x < 4; x++ {
		binary[3*w+x] = true
	}

	rect := largestComponentBounds(binary, w, h)
	want := image.Rect(0, 0, 4, 4)
	if rect != want {
		t.Errorf("largestComponentBounds = %v, want %v", rect, want)
	}
}

func TestLargestComponentBounds_NoForeground(t *testing.T) {
	binary := make([]bool, 25)
	rect := largestComponentBounds(binary, 5, 5)
	if !rect.Empty() {
		t.Errorf("largestComponentBounds = %v, want empty", rect)
	}
}
