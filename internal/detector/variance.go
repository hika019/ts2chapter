package detector

import (
	"image"
)

const varianceHistogramBins = 256

// VarianceMap holds per-pixel mean and variance computed over multiple frames.
type VarianceMap struct {
	Mean     [][]float64
	Variance [][]float64
	Width    int
	Height   int
}

// BuildVarianceMap computes per-pixel mean and variance from grayscale frames
// using Welford's online algorithm (single-pass, numerically stable).
func BuildVarianceMap(frames []image.Gray) *VarianceMap {
	if len(frames) == 0 {
		return &VarianceMap{}
	}

	w := frames[0].Bounds().Dx()
	h := frames[0].Bounds().Dy()

	// Flat arrays for cache-friendly access in the hot loop.
	mean := make([]float64, h*w)
	m2 := make([]float64, h*w)

	for n := 0; n < len(frames); n++ {
		count := float64(n + 1)
		pix := frames[n].Pix
		stride := frames[n].Stride
		for y := 0; y < h; y++ {
			rowPix := y * stride
			rowAcc := y * w
			for x := 0; x < w; x++ {
				idx := rowAcc + x
				val := float64(pix[rowPix+x])
				delta := val - mean[idx]
				mean[idx] += delta / count
				delta2 := val - mean[idx]
				m2[idx] += delta * delta2
			}
		}
	}

	nf := float64(len(frames))
	meanMap := make([][]float64, h)
	varMap := make([][]float64, h)
	for y := 0; y < h; y++ {
		meanMap[y] = make([]float64, w)
		varMap[y] = make([]float64, w)
		off := y * w
		for x := 0; x < w; x++ {
			meanMap[y][x] = mean[off+x]
			varMap[y][x] = m2[off+x] / nf
		}
	}

	return &VarianceMap{
		Mean:     meanMap,
		Variance: varMap,
		Width:    w,
		Height:   h,
	}
}

// OtsuThreshold computes the optimal threshold for a histogram using Otsu's method.
// Returns the bin index (as float64) that maximizes between-class variance.
func OtsuThreshold(histogram []int) float64 {
	total := 0
	sumAll := 0.0
	for i, count := range histogram {
		total += count
		sumAll += float64(i) * float64(count)
	}
	if total == 0 {
		return 0
	}

	sumB := 0.0
	wB := 0
	maxSigma := 0.0
	threshold := 0

	for t, count := range histogram {
		wB += count
		if wB == 0 {
			continue
		}
		wF := total - wB
		if wF == 0 {
			break
		}
		sumB += float64(t) * float64(count)
		mB := sumB / float64(wB)
		mF := (sumAll - sumB) / float64(wF)
		sigma := float64(wB) * float64(wF) * (mB - mF) * (mB - mF)
		if sigma > maxSigma {
			maxSigma = sigma
			threshold = t
		}
	}

	return float64(threshold)
}

// DetectLogoRegion finds the logo region by applying OTSU thresholding to the
// variance histogram, then finding the largest connected component of low-variance pixels.
func (vm *VarianceMap) DetectLogoRegion() image.Rectangle {
	if vm.Width == 0 || vm.Height == 0 {
		return image.Rectangle{}
	}

	maxVar := 0.0
	for y := 0; y < vm.Height; y++ {
		for x := 0; x < vm.Width; x++ {
			if vm.Variance[y][x] > maxVar {
				maxVar = vm.Variance[y][x]
			}
		}
	}
	if maxVar == 0 {
		return image.Rectangle{}
	}

	// Build variance histogram with linear binning.
	hist := make([]int, varianceHistogramBins)
	binWidth := maxVar / float64(varianceHistogramBins-1)
	for y := 0; y < vm.Height; y++ {
		for x := 0; x < vm.Width; x++ {
			bin := int(vm.Variance[y][x] / binWidth)
			if bin >= varianceHistogramBins {
				bin = varianceHistogramBins - 1
			}
			hist[bin]++
		}
	}

	threshBin := int(OtsuThreshold(hist))

	// Binary mask: low variance (bin ≤ threshold) = logo candidate.
	binary := make([]bool, vm.Height*vm.Width)
	for y := 0; y < vm.Height; y++ {
		for x := 0; x < vm.Width; x++ {
			bin := int(vm.Variance[y][x] / binWidth)
			if bin >= varianceHistogramBins {
				bin = varianceHistogramBins - 1
			}
			binary[y*vm.Width+x] = bin <= threshBin
		}
	}

	return largestComponentBounds(binary, vm.Width, vm.Height)
}

// --- Union-Find for connected component labeling ---

type unionFind struct {
	parent []int
}

func newUnionFind() *unionFind {
	return &unionFind{parent: []int{0}} // index 0 reserved (background)
}

func (uf *unionFind) makeSet() int {
	n := len(uf.parent)
	uf.parent = append(uf.parent, n)
	return n
}

func (uf *unionFind) find(x int) int {
	for uf.parent[x] != x {
		uf.parent[x] = uf.parent[uf.parent[x]] // path halving
		x = uf.parent[x]
	}
	return x
}

func (uf *unionFind) union(a, b int) {
	ra, rb := uf.find(a), uf.find(b)
	if ra != rb {
		uf.parent[rb] = ra
	}
}

// largestComponentBounds performs 2-pass connected component labeling (4-connectivity)
// and returns the bounding box of the largest foreground component.
func largestComponentBounds(binary []bool, w, h int) image.Rectangle {
	labels := make([]int, h*w)
	uf := newUnionFind()

	// Pass 1: assign provisional labels.
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*w + x
			if !binary[idx] {
				continue
			}

			left, above := 0, 0
			if x > 0 && binary[idx-1] {
				left = labels[idx-1]
			}
			if y > 0 && binary[idx-w] {
				above = labels[idx-w]
			}

			switch {
			case left == 0 && above == 0:
				labels[idx] = uf.makeSet()
			case left != 0 && above == 0:
				labels[idx] = left
			case left == 0 && above != 0:
				labels[idx] = above
			default:
				minL := left
				if above < left {
					minL = above
				}
				labels[idx] = minL
				uf.union(left, above)
			}
		}
	}

	// Pass 2: resolve labels and count component sizes.
	counts := make(map[int]int)
	for i := range labels {
		if labels[i] != 0 {
			labels[i] = uf.find(labels[i])
			counts[labels[i]]++
		}
	}

	if len(counts) == 0 {
		return image.Rectangle{}
	}

	// Find the largest component.
	maxLabel, maxCount := 0, 0
	for label, count := range counts {
		if count > maxCount {
			maxCount = count
			maxLabel = label
		}
	}

	// Compute bounding box.
	minX, minY := w, h
	maxX, maxY := 0, 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if labels[y*w+x] == maxLabel {
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}

	return image.Rect(minX, minY, maxX+1, maxY+1)
}
