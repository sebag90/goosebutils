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

func collectLineResult(line string, indeces []int, lineNum, lineOffset, windowSize int) []string {
	results := []string{}

	start, end := indeces[0], indeces[1]
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

func processChunk(
	inputString,
	rightExtension string,
	searchPattern *regexp.Regexp,
	currentLine, currentCharPosition int,
) ([]string, int, int) {
	var results []string
	processedChars := currentCharPosition
	splitted := strings.Split(inputString, "\n")

	for lineIndex, line := range splitted {
		var rightExtendedMargin string
		if lineIndex == len(splitted)-1 {
			rightExtendedMargin = strings.Split(rightExtension, "\n")[0]
			line += rightExtendedMargin
		}
		resultIndeces := searchPattern.FindAllStringIndex(line, -1)

		for _, indeces := range resultIndeces {
			if indeces[0] > len(line)-len(rightExtendedMargin) {
				continue
			}
			searchResults := collectLineResult(line, indeces, currentLine+lineIndex, processedChars, windowSize)
			results = append(results, searchResults...)

		}

		if lineIndex == len(splitted)-1 {
			processedChars += len(line) - len(rightExtendedMargin)
		} else {
			processedChars = 1
		}

	}
	return results, len(splitted) - 1, processedChars
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

	const chunkSize = 1024
	const nextChunkSize = 512
	var n int

	nextBuffer := make([]byte, chunkSize)
	buffer := make([]byte, chunkSize)
	var fileResults, results []string

	currentLine := 1
	var prevN, processedChars, processedLines int

	for {
		// schedule next read
		n, err = file.Read(nextBuffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return
		}

		// process previous buffer
		thisChunk := buffer[:prevN] // prevN is the previous read size
		if len(thisChunk) > 0 {
			rightExtension := nextBuffer[:Min(n, nextChunkSize)]
			results, processedLines, processedChars = processChunk(
				string(thisChunk),
				string(rightExtension),
				searchPattern,
				currentLine,
				processedChars,
			)
			fileResults = append(fileResults, results...)
			currentLine += processedLines
		}

		// rotate buffers and sizes
		buffer, nextBuffer = nextBuffer, buffer
		prevN = n
	}

	// process last chunk
	results, _, _ = processChunk(
		string(buffer[:prevN]),
		"",
		searchPattern,
		currentLine,
		processedChars,
	)
	fileResults = append(fileResults, results...)

	if !utf8.Valid(buffer) {
		return
	}

	if len(fileResults) > 0 {
		printResult(filePath, fileResults)
	}
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
