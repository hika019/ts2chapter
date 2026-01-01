package cmd

import (
	"fmt"
	"os"

	"github.com/hika019/ts2chapter.git/internal/detector"

	"github.com/spf13/cobra"
)

var logoCmd = &cobra.Command{
	Use:   "logo-detect [TSファイル]",
	Short: "放送局のロゴの有無を元にチャプターを生成（CM検出）",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		tsFile := args[0]
		err := detector.GenerateChaptersByLogo(tsFile)
		if err != nil {
			fmt.Printf("エラー: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("ロゴ検出によるチャプター生成が完了しました。")
	},
}

func init() {
	rootCmd.AddCommand(logoCmd)
}
