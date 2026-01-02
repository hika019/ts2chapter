package cmd

import (
	"os"

	// Import the detector package

	"github.com/hika019/ts2chapter.git/internal/detector"
	"github.com/spf13/cobra"
)

var debugLog bool

var rootCmd = &cobra.Command{
	Use:   "ts2chapter",
	Short: "TSファイルからチャプター情報を生成するツール",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		detector.InitLogger(debugLog)
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(
		&debugLog, "debug", "d", false, "Enable debug logging",
	)
}
