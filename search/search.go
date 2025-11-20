package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"unicode/utf8"
)

const (
	PURPLE = "\u001b[95m"
	GREEN  = "\u001b[92m"
	RED    = "\u001b[91m"
	END    = "\u001b[0m"
	YELLOW = "\u001b[93m"
)

var printMutex sync.Mutex

func Min(a, b int) int {
	if a > b {
		return b
	}
	return a
}

func isValidPath(path string, info os.FileInfo, filePattern *regexp.Regexp, excludePattern []*regexp.Regexp) bool {
	if info.IsDir() {
		return false
	}

	for _, pattern := range excludePattern {
		if pattern.MatchString(path) {
			return false
		}
	}

	if !filePattern.MatchString(path) {
		return false
	}

	return true
}

func collectPaths(root string, pattern *regexp.Regexp, excludePattern []*regexp.Regexp, jobs chan<- string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if isValidPath(path, info, pattern, excludePattern) {
			jobs <- path
		}
		return nil
	})
}

func collectLineResult(line string, indeces [][]int, lineNum, lineOffset, windowSize int) []string {
	results := []string{}

	for _, m := range indeces {
		start, end := m[0], m[1]
		leftMarginIndex := max(0, start-windowSize)
		rightMarginIndex := min(len(line), end+windowSize)
		if windowSize < 0 {
			leftMarginIndex = 0
			rightMarginIndex = len(line)
		}

		leftMargin := line[leftMarginIndex:start]
		rightMargin := line[end:rightMarginIndex]
		coloredWord := fmt.Sprintf("%s%s%s", RED, line[start:end], END)
		linetoDisplay := fmt.Sprintf("%s%s%s", leftMargin, coloredWord, rightMargin)

		results = append(results, fmt.Sprintf("\t%s:%-20s%s",
			fmt.Sprintf("%s%d%s", YELLOW, lineNum, END),
			fmt.Sprintf("%s%d%s", GREEN, start+lineOffset, END),
			strings.TrimSpace(linetoDisplay),
		))
	}
	return results
}

func printResult(fileName string, results []string) {
	printMutex.Lock()
	fmt.Printf("%s%s%s\n", PURPLE, fileName, END)
	for _, line := range results {
		fmt.Println(line)
	}
	printMutex.Unlock()
}

func findFileResults(thisLine []byte, searchPattern *regexp.Regexp, lineNum, lineOffset, windowSize int) []string {
	searchResults := []string{}
	toProcess := string(thisLine)
	indeces := searchPattern.FindAllStringIndex(toProcess, -1)

	if indeces != nil {
		searchResults = collectLineResult(toProcess, indeces, lineNum, lineOffset, windowSize)
	}
	return searchResults
}

func searchInFile(filePath string, searchPattern *regexp.Regexp, windowSize int, nameOnly bool) {
	if nameOnly {
		printResult(filePath, []string{})
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	const chunkSize = 10 //1024 * 1024
	const nextChunkSize = 512
	var n int

	nextBuffer := make([]byte, chunkSize)
	buffer := make([]byte, chunkSize)

	// fileResults := []string{}
	// lineNum := 1
	// lineOffset := 1
	var prevN int
	for {
		// schedule next read
		n, err = file.Read(nextBuffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return
		}

		// process *previous* buffer
		thisChunk := buffer[:prevN] // prevN is the previous read size
		if len(thisChunk) > 0 {
			fmt.Println(string(thisChunk), "--", string(nextBuffer[:n]))
		}

		// rotate buffers and sizes
		buffer, nextBuffer = nextBuffer, buffer
		prevN = n
	}

	if !utf8.Valid(buffer) {
		return
	}

	// 	fmt.Println(string(thisChunk))

	// 	thisLine := []byte{}

	// 	for _, letter := range thisChunk {
	// 		if letter == '\n' {
	// 			lineResults := findFileResults(thisLine, searchPattern, lineNum, lineOffset, windowSize)
	// 			fileResults = append(fileResults, lineResults...)
	// 			// update counters
	// 			lineNum++
	// 			lineOffset = 1
	// 			thisLine = []byte{}
	// 			continue
	// 		}
	// 		thisLine = append(thisLine, letter)
	// 	}

	// 	// the rest will be processed in next iteration, remove it from lineOffset
	// 	lineOffset += len(thisChunk) - len(thisLine)

	// 	buffer = nextBuffer
	// 	fmt.Println(string(buffer), "--")
	// }

	// if len(fileResults) > 0 {
	// 	printResult(filePath, fileResults)
	// }
}

func Search(path, searchPattern, filePattern string, excludeFilePatterns []string, windowSize int, ignoreCase, nameOnly bool) {
	if ignoreCase {
		filePattern = "(?i)" + filePattern
		searchPattern = "(?i)" + searchPattern
	}

	fileRegex := regexp.MustCompile(filePattern)
	searchRegex := regexp.MustCompile(searchPattern)
	excludeRegex := []*regexp.Regexp{}
	for _, pattern := range excludeFilePatterns {
		excludeRegex = append(excludeRegex, regexp.MustCompile(pattern))
	}

	numWorkers := runtime.NumCPU() * 2
	runtime.GOMAXPROCS(numWorkers)

	var wg sync.WaitGroup
	wg.Add(numWorkers)
	jobs := make(chan string, 100)

	// start consumers
	for range numWorkers {
		go func() {
			defer wg.Done()
			for file := range jobs {
				searchInFile(file, searchRegex, windowSize, nameOnly)
			}
		}()
	}

	// start producers
	go func() {
		collectPaths(path, fileRegex, excludeRegex, jobs)
		close(jobs)
	}()

	wg.Wait()
}
