package detector

import (
	"math"
	"testing"
)

func TestMedianFilterBool(t *testing.T) {
	tests := []struct {
		name       string
		series     []bool
		windowSize int
		expected   []bool
	}{
		{
			name:       "empty series",
			series:     []bool{},
			windowSize: 3,
			expected:   []bool{},
		},
		{
			name:       "single element",
			series:     []bool{true},
			windowSize: 3,
			expected:   []bool{true},
		},
		{
			name:       "window size 1",
			series:     []bool{true, false, true, false},
			windowSize: 1,
			expected:   []bool{true, false, true, false},
		},
		{
			name:       "remove single false spike",
			series:     []bool{true, true, false, true, true},
			windowSize: 3,
			expected:   []bool{true, true, true, true, true},
		},
		{
			name:       "remove single true spike",
			series:     []bool{false, false, true, false, false},
			windowSize: 3,
			expected:   []bool{false, false, false, false, false},
		},
		{
			name:       "window size 5",
			series:     []bool{true, true, false, false, true, true, true},
			windowSize: 5,
			expected:   []bool{true, true, true, true, true, true, true},
		},
		{
			name:       "window larger than series",
			series:     []bool{true, false, true},
			windowSize: 10,
			expected:   []bool{true, true, true},
		},
		{
			name:       "zero window size (defaults to 1)",
			series:     []bool{true, false, true},
			windowSize: 0,
			expected:   []bool{true, false, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MedianFilterBool(tt.series, tt.windowSize)
			if len(result) != len(tt.expected) {
				t.Errorf("length mismatch: got %d, want %d", len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("index %d: got %v, want %v", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestMedianFilterFloat(t *testing.T) {
	tests := []struct {
		name       string
		series     []float64
		windowSize int
		expected   []float64
		tolerance  float64
	}{
		{
			name:       "empty series",
			series:     []float64{},
			windowSize: 3,
			expected:   []float64{},
			tolerance:  1e-9,
		},
		{
			name:       "single element",
			series:     []float64{5.0},
			windowSize: 3,
			expected:   []float64{5.0},
			tolerance:  1e-9,
		},
		{
			name:       "window size 1",
			series:     []float64{1.0, 2.0, 3.0, 4.0},
			windowSize: 1,
			expected:   []float64{1.0, 2.0, 3.0, 4.0},
			tolerance:  1e-9,
		},
		{
			name:       "remove spike in middle",
			series:     []float64{1.0, 1.0, 10.0, 1.0, 1.0},
			windowSize: 3,
			expected:   []float64{1.0, 1.0, 1.0, 1.0, 1.0},
			tolerance:  1e-9,
		},
		{
			name:       "smooth transition",
			series:     []float64{1.0, 2.0, 3.0, 4.0, 5.0},
			windowSize: 3,
			expected:   []float64{1.5, 2.0, 3.0, 4.0, 4.5},
			tolerance:  1e-9,
		},
		{
			name:       "window size 5",
			series:     []float64{5.0, 3.0, 1.0, 2.0, 4.0, 6.0, 7.0},
			windowSize: 5,
			expected:   []float64{3.0, 2.5, 3.0, 3.0, 4.0, 5.0, 6.0},
			tolerance:  1e-9,
		},
		{
			name:       "negative values",
			series:     []float64{-1.0, -2.0, -3.0, -2.0, -1.0},
			windowSize: 3,
			expected:   []float64{-1.5, -2.0, -2.0, -2.0, -1.5},
			tolerance:  1e-9,
		},
		{
			name:       "window larger than series",
			series:     []float64{1.0, 5.0, 3.0},
			windowSize: 10,
			expected:   []float64{3.0, 3.0, 4.0},
			tolerance:  1e-9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MedianFilterFloat(tt.series, tt.windowSize)
			if len(result) != len(tt.expected) {
				t.Errorf("length mismatch: got %d, want %d", len(result), len(tt.expected))
				return
			}
			for i := range result {
				if math.Abs(result[i]-tt.expected[i]) > tt.tolerance {
					t.Errorf("index %d: got %f, want %f", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestMedian(t *testing.T) {
	tests := []struct {
		name      string
		values    []float64
		expected  float64
		tolerance float64
	}{
		{
			name:      "empty slice",
			values:    []float64{},
			expected:  0.0,
			tolerance: 1e-9,
		},
		{
			name:      "single value",
			values:    []float64{5.0},
			expected:  5.0,
			tolerance: 1e-9,
		},
		{
			name:      "odd length",
			values:    []float64{1.0, 3.0, 5.0},
			expected:  3.0,
			tolerance: 1e-9,
		},
		{
			name:      "even length",
			values:    []float64{1.0, 2.0, 3.0, 4.0},
			expected:  2.5,
			tolerance: 1e-9,
		},
		{
			name:      "unsorted input",
			values:    []float64{5.0, 1.0, 3.0, 2.0, 4.0},
			expected:  3.0,
			tolerance: 1e-9,
		},
		{
			name:      "negative values",
			values:    []float64{-3.0, -1.0, -2.0},
			expected:  -2.0,
			tolerance: 1e-9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := median(tt.values)
			if math.Abs(result-tt.expected) > tt.tolerance {
				t.Errorf("got %f, want %f", result, tt.expected)
			}
		})
	}
}
