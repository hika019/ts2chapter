/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/hika019/ts2chapter.git/internal/detector"
	"github.com/spf13/cobra"
)

var sCmd = &cobra.Command{
	Use:   "silence [TSファイル]",
	Short: "無音区間を使ってチャプターを分割",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tsPath := args[0]
		detector.InitLogger(debugLog)

		if err := detector.GenerateChaptersBySilence(tsPath); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(sCmd)
}
