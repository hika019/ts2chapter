package cmd

import (
	"fmt"

	"github.com/hika019/ts2chapter.git/internal/detector"
	"github.com/spf13/cobra"
)

var combinedThreshold float32

var combinedCmd = &cobra.Command{
	Use:   "combined [TSファイル]",
	Short: "無音検出とロゴ検出を組み合わせた高精度チャプター生成",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if combinedThreshold < 0.0 || combinedThreshold > 1.0 {
			return fmt.Errorf(
				"--main-threshold must be between 0.0 and 1.0 (got %.3f)",
				combinedThreshold,
			)
		}

		tsFile := args[0]
		return detector.GenerateChaptersByCombined(tsFile, combinedThreshold)
	},
}

func init() {
	rootCmd.AddCommand(combinedCmd)
	combinedCmd.Flags().Float32Var(
		&combinedThreshold,
		"main-threshold",
		0.55,
		"本編検出の一致率しきい値 (0.0 to 1.0)",
	)
}
