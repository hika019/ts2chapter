package cmd

import (
	"fmt"

	"github.com/hika019/ts2chapter.git/internal/detector"

	"github.com/spf13/cobra"
)

var mainThreshold float32

var logoCmd = &cobra.Command{
	Use:   "logo [TSファイル]",
	Short: "放送局のロゴの有無を元にチャプターを生成（CM検出）",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if mainThreshold < 0.0 || mainThreshold > 1.0 {
			return fmt.Errorf(
				"--main-threshold must be between 0.0 and 1.0 (got %.3f)",
				mainThreshold,
			)
		}

		tsFile := args[0]
		return detector.GenerateChaptersByLogo(tsFile, mainThreshold)
	},
}

func init() {
	rootCmd.AddCommand(logoCmd)
	logoCmd.Flags().Float32Var(
		&mainThreshold,
		"main-threshold",
		0.55,
		"本編検出の一致率しきい値 (0.0 to 1.0)",
	)
}
