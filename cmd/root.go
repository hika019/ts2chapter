package cmd

import (
	"fmt"
	"os"

	// Import the detector package
	"github.com/hika019/ts2chapter.git/internal/detector"
	"github.com/spf13/cobra"
)

var silenceDetect bool

var rootCmd = &cobra.Command{
	Use:   "ts2chapter [TSファイル]",
	Short: "放送波由来TSファイルからチャプター情報を生成",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		tsPath := args[0]
		if silenceDetect {
			err := detector.GenerateChapters(tsPath)
			if err != nil {
				fmt.Printf("エラー: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("チャプター情報を生成しました。")
		}
	},
}

func Execute() {
	rootCmd.Flags().BoolVar(&silenceDetect, "silence-detect", false, "無音区間を使ってチャプターを分割")
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
