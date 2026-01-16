package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const maxTokenSize = 10 * 1024 * 1024 // 10 MB

func main() {
	// Use official nuclei-templates as default
	fileFlag := flag.String("file", "./temp/nuclei-templates", "Root directory for nuclei templates")
	outputDirFlag := flag.String("output-directory", "./technologies/nuclei-technologies", "Directory to save the output files")
	flag.Parse()

	rootDir := *fileFlag
	outputDir := *outputDirFlag

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory '%s': %v", outputDir, err)
	}

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".yaml" {
			processYAMLFile(path, outputDir)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Error walking through directory: %v", err)
	}

	fmt.Println("Running anew to create *-all.txt file for every directory...")
	directories := collectDirs(outputDir)
	for _, dir := range directories {
		if dir == "" || dir == "." {
			continue
		}
		cmdStr := fmt.Sprintf("cat %s/*-*.txt | anew %s/all.txt", dir, dir)
		cmd := exec.Command("sh", "-c", cmdStr)
		if err := cmd.Run(); err != nil {
			log.Printf("Failed to execute anew for directory %s: %v", dir, err)
		}
	}
	fmt.Println("Finished.")
}

func collectDirs(root string) []string {
	var dirs []string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() {
			dirs = append(dirs, path)
		}
		return nil
	})
	return dirs
}

func processYAMLFile(templateFile, outputDir string) {
	file, err := os.Open(templateFile)
	if err != nil {
		log.Printf("Failed to open file: %v", err)
		return
	}
	defer file.Close()

	var tags, severity string
	var baseURLs []string
	tagsRegex := regexp.MustCompile(`^ *tags: *(.+)$`)
	severityRegex := regexp.MustCompile(`^ *severity: *(.+)$`)
	baseURLRegex := regexp.MustCompile(`- ['"]{{BaseURL}}([^'"]*)['"]`)

	scanner := bufio.NewScanner(file)
	buf := make([]byte, maxTokenSize)
	scanner.Buffer(buf, maxTokenSize)

	for scanner.Scan() {
		line := scanner.Text()
		if matches := tagsRegex.FindStringSubmatch(line); len(matches) > 0 {
			tags = matches[1]
		}
		if matches := severityRegex.FindStringSubmatch(line); len(matches) > 0 {
			severity = matches[1]
		}
		if matches := baseURLRegex.FindStringSubmatch(line); len(matches) > 0 {
			baseURLs = append(baseURLs, matches[1])
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("Error reading file: %v", err)
		return
	}

	tagList := strings.Split(tags, ",")
	validTagRegex := regexp.MustCompile(`^[a-zA-Z0-9-_]+$`)
	       severityMapping := map[string]string{
		       "none": "unknown",
		       "informative": "info",
		       "medium": "medium",
		       "high": "high",
		       "critical": "critical",
	       }
	validSeverities := map[string]bool{
		"unknown": true, "info": true, "low": true, "medium": true, "high": true, "critical": true,
	}
	if corrected, ok := severityMapping[strings.ToLower(severity)]; ok {
		severity = corrected
	}
	if !validSeverities[strings.ToLower(severity)] {
		return
	}
	if len(baseURLs) == 0 {
		return
	}
	baseFilename := filepath.Base(templateFile)
	for _, tag := range tagList {
		tag = strings.TrimSpace(tag)
		if tag == "" || !validTagRegex.MatchString(tag) {
			continue
		}
		tagDir := filepath.Join(outputDir, tag)
		if err := os.MkdirAll(tagDir, 0755); err != nil {
			log.Printf("Failed to create directory '%s': %v", tagDir, err)
			continue
		}
		lowercaseSeverity := strings.ToLower(severity)
		outputFile := filepath.Join(tagDir, fmt.Sprintf("%s-%s.txt", tag, lowercaseSeverity))
		cmd := exec.Command("anew", outputFile)
		cmd.Stdin = strings.NewReader(strings.Join(baseURLs, "\n"))
		if err := cmd.Run(); err != nil {
			log.Printf("Failed to execute anew: %v", err)
			continue
		}
		fmt.Printf("SOURCE:%s => DESTINATION:%s\n", baseFilename, outputFile)
	}
}
