package cmd

import (
	"os"

	// Import the detector package

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ts2chapter",
	Short: "TSファイルからチャプター情報を生成するツール",
	Long:  `TSファイルから無音区間やロゴの有無を検出してチャプター情報を生成するツールです。`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
