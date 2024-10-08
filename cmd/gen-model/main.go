package main

import (
	"github.com/spf13/cobra"
	"github.com/withz/ptun/pkg/proto"
)

var (
	rootCmd = &cobra.Command{
		Use:   "run",
		Short: "Gen code from given file to output path",
		Run:   Root,
	}

	source string
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&source, "source", "s", "model", "source file")
}

func Root(cmd *cobra.Command, args []string) {
	err := proto.GenMessageMethod(source)
	if err != nil {
		panic(err)
	}
}
func main() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
