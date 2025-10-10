package main

import (
	"bufio"
	"fmt"
	"log"
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

func collectPaths(root string, pattern *regexp.Regexp, excludePattern []*regexp.Regexp) ([]string, error) {
	files := []string{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if isValidPath(path, info, pattern, excludePattern) {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

func collectResult(line string, indeces [][]int, lineNum, windowSize int) []string {
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

		results = append(results, fmt.Sprintf("\t%s:%s\t%s",
			fmt.Sprintf("%s%d%s", YELLOW, lineNum, END),
			fmt.Sprintf("%s%d%s", GREEN, start, END),
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

	scanner := bufio.NewScanner(file)
	const maxCapacity = 1024 * 1024 * 100
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, maxCapacity)

	fileResults := []string{}
	lineIndex := 1
	for scanner.Scan() {
		bytesLine := scanner.Bytes()
		if !utf8.Valid(bytesLine) {
			return
		}
		line := string(bytesLine)
		indeces := searchPattern.FindAllStringIndex(line, -1)
		if indeces != nil {
			lineResult := collectResult(line, indeces, lineIndex, windowSize)
			fileResults = append(fileResults, lineResult...)
		}
		lineIndex++
	}

	if len(fileResults) > 0 {
		printResult(filePath, fileResults)
	}

	if err := scanner.Err(); err != nil {
		return
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
	// excludePattern := regexp.MustCompile(excludeFilePattern)
	for _, pattern := range excludeFilePatterns {
		excludeRegex = append(excludeRegex, regexp.MustCompile(pattern))
	}

	files, err := collectPaths(path, fileRegex, excludeRegex)
	if err != nil {
		log.Printf("error walking the path: %v", err)
	}
	numWorkers := runtime.NumCPU() * 2
	runtime.GOMAXPROCS(numWorkers)

	var wg sync.WaitGroup
	jobs := make(chan string, numWorkers*2)

	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range jobs {
				searchInFile(filePath, searchRegex, windowSize, nameOnly)
			}
		}()
	}

	for _, filePath := range files {
		jobs <- filePath
	}

	close(jobs)
	wg.Wait()
}
