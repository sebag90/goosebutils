package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	path                string
	filePattern         string
	excludeFilePatterns []string
	windowSize          int
	ignoreCase          bool
	nameOnly            bool
)

var rootCmd = &cobra.Command{
	Use:   "search [pattern]",
	Short: "Search text within files",
	Long:  "Search for a regex pattern in all the files inside a directory and its subdirectories",
	Args:  cobra.ExactArgs(1),
	Run:   startSearch,
}

func init() {
	rootCmd.Flags().StringVarP(&path, "path", "p", ".", "Directory to search")
	rootCmd.Flags().StringVarP(&filePattern, "file-regex", "f", ".", "regex pattern to filter files")
	rootCmd.Flags().StringSliceVarP(&excludeFilePatterns, "exclude-file", "e", []string{"a^"}, "regex pattern to exclude files from search")
	rootCmd.Flags().IntVarP(&windowSize, "winsize", "w", 10, "Size of the print window")
	rootCmd.Flags().BoolVarP(&ignoreCase, "ignore-case", "i", false, "ignore case")
	rootCmd.Flags().BoolVarP(&nameOnly, "name-only", "n", false, "match only the name of the file (path included)")
}

func startSearch(cmd *cobra.Command, args []string) {
	if nameOnly {
		if filePattern != "." {
			fmt.Println("When searching file names only, the main regex pattern will be used")
		}
		filePattern = args[0]
	}

	searchPattern := args[0]

	Search(path, searchPattern, filePattern, excludeFilePatterns, windowSize, ignoreCase, nameOnly)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
