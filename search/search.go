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

	const chunkSize = 1024 * 1024
	buffer := make([]byte, chunkSize)

	fileResults := []string{}
	lineNum := 1
	lineOffset := 0
	var carryOver []byte

	for {
		_, err := file.Read(buffer)

		if !utf8.Valid(buffer) {
			return
		}

		// carry over line index and offset from last iteration
		startLineOffset := lineOffset
		startLineNum := lineNum

		// init variables to track upcoming line
		currentLine := true
		thisChunk := append(carryOver, buffer...)
		carryOver = []byte{}
		toProcess := []byte{}

		for _, letter := range thisChunk {
			if letter == '\n' {
				lineNum++
				lineOffset = 0
				currentLine = false
				continue
			}

			if currentLine == true {
				lineOffset++
				toProcess = append(toProcess, letter)
			} else {
				carryOver = append(carryOver, letter)

			}

		}

		if err != nil {
			if err == io.EOF {
				break
			}
			return
		}

		line := string(toProcess)
		indeces := searchPattern.FindAllStringIndex(line, -1)

		if indeces != nil {
			lineResult := collectLineResult(line, indeces, startLineNum, startLineOffset, windowSize)
			fileResults = append(fileResults, lineResult...)
		}
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
