package detector

import (
	"sort"
)

// MedianFilterBool applies a median filter to a boolean time series.
// It converts bool to 0/1, computes the median over a sliding window,
// and thresholds back to bool (median >= 0.5 → true).
// Edge values are handled by zero-padding.
func MedianFilterBool(series []bool, windowSize int) []bool {
	if len(series) == 0 {
		return []bool{}
	}
	if windowSize <= 0 {
		windowSize = 1
	}
	if windowSize > len(series) {
		windowSize = len(series)
	}

	// Convert bool to float for median computation
	floatSeries := make([]float64, len(series))
	for i, v := range series {
		if v {
			floatSeries[i] = 1.0
		} else {
			floatSeries[i] = 0.0
		}
	}

	// Apply float median filter
	filtered := MedianFilterFloat(floatSeries, windowSize)

	// Threshold back to bool
	result := make([]bool, len(filtered))
	for i, v := range filtered {
		result[i] = v >= 0.5
	}

	return result
}

// MedianFilterFloat applies a median filter to a float64 time series.
// It uses a sliding window of the specified size.
// Edge values are handled by zero-padding (symmetric extension at boundaries).
func MedianFilterFloat(series []float64, windowSize int) []float64 {
	if len(series) == 0 {
		return []float64{}
	}
	if windowSize <= 0 {
		windowSize = 1
	}
	if windowSize > len(series) {
		windowSize = len(series)
	}

	result := make([]float64, len(series))
	halfWindow := windowSize / 2

	for i := range series {
		// Collect window values
		window := make([]float64, 0, windowSize)

		for j := i - halfWindow; j <= i+halfWindow; j++ {
			if j >= 0 && j < len(series) {
				window = append(window, series[j])
			}
		}

		// Compute median
		result[i] = median(window)
	}

	return result
}

// median computes the median of a slice.
// For even-length slices, returns the average of the two middle elements.
func median(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	// Create a copy to avoid modifying the input
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2.0
}
