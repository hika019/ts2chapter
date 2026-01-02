/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/hika019/ts2chapter.git/internal/detector"
	"github.com/spf13/cobra"
)

var sCmd = &cobra.Command{
	Use:   "silence [TSファイル]",
	Short: "無音区間を使ってチャプターを分割",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		tsPath := args[0]
		fmt.Println("無音区間を使ってチャプターを分割します...")
		err := detector.GenerateChapters(tsPath)
		if err != nil {
			fmt.Printf("エラー: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("チャプター情報を生成しました。")
	},
}

func init() {
	rootCmd.AddCommand(sCmd)
}
