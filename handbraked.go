package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/howeyc/fsnotify"
	"github.com/tangerilli/handbraked/common"
	"github.com/tangerilli/handbraked/ui"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var movieExtensions []string = []string{"m4v", "avi", "mpg", "mpeg", "mkv", "mov"}

func usage() {
	fmt.Println("usage: handbraked [path to watch] [output path] [movie file path]")
	flag.PrintDefaults()
	os.Exit(1)
}

func move(oldPath string, newPath string) error {
	oldFile, err := os.Open(oldPath)
	if err != nil {
		return err
	}
	defer oldFile.Close()

	newFile, err := os.Create(newPath)
	if err != nil {
		return err
	}
	defer newFile.Close()

	_, err = io.Copy(oldFile, newFile)
	if err != nil {
		return err
	}
	newFile.Sync()

	os.Remove(oldPath)

	return nil
}

func processFile(path string, outputPath string, c chan float64) {
	defer close(c)

	fmt.Println("Processing " + path)
	newFileName := filepath.Base(path)
	newFileName = newFileName[:len(newFileName)-3] + "m4v"
	outputFile := filepath.Join(outputPath, newFileName)
	tmpOutputFile := filepath.Join("/tmp", newFileName)
	cmd := exec.Command("HandBrakeCLI", "-i", path, "--preset=iPad", "-o", tmpOutputFile)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("Error opening stdout")
		return
	}

	go func() {
		b := make([]byte, 1024)
		pattern, err := regexp.Compile("\\d+\\.\\d+")
		if err != nil {
			fmt.Println("Error compiling pattern")
		}
		for l, err := stdout.Read(b); l > 0 || err == nil; l, err = stdout.Read(b) {
			if l > 0 {
				percentageStr := pattern.FindString(string(b))
				percentage, err := strconv.ParseFloat(percentageStr, 32)
				if err == nil {
					c <- percentage
				}
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		fmt.Println("Error starting")
		return
	}

	if err := cmd.Wait(); err != nil {
		fmt.Println("Error waiting")
		return
	}
	fmt.Printf("Renaming %s to %s\n", tmpOutputFile, outputFile)
	err = os.Rename(tmpOutputFile, outputFile)
	if err != nil {
		fmt.Println("Error renaming file: ", err.Error())
		fmt.Println("Copying file instead")
		err = move(tmpOutputFile, outputFile)
		if err != nil {
			fmt.Println("Error copying file: ", err.Error())
		}
	}
}

type StatusUpdate struct {
	Name     string
	Progress float64
}

func HandleFile(file string, outputPath string, deleteOnCompletion bool) {
	c := make(chan float64)
	go processFile(file, outputPath, c)
	status := StatusUpdate{filepath.Base(file), 0}
	last := time.Now().Unix()
	for output := range c {
		fmt.Println(output)
		now := time.Now().Unix()
		if now-last < 2 {
			continue
		}
		last = time.Now().Unix()

		status.Progress = output
		j, err := json.Marshal(status)
		if err != nil {
			continue
		}
		ui.MessageHub.Broadcast <- string(j)
	}
	fmt.Println("Finished " + file)
	if deleteOnCompletion {
		os.Remove(file)
	}
}

func main() {
	flag.Usage = usage
	var deleteOnCompletion = flag.Bool("delete", true, "Delete files after processing")
	var staticPath = flag.String("static", "/static/", "Path to directory containing static website files")
	flag.Parse()
	args := flag.Args()
	if len(args) < 3 {
		usage()
	}
	inputPath := args[0]
	outputPath := args[1]
	moviePath := args[2]

	if moviePath[len(moviePath)-1] == "/"[0] {
		moviePath = moviePath[:len(moviePath)-1]
	}

	// Sorted so it can be searched later
	sort.Strings(movieExtensions)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			select {
			case ev := <-watcher.Event:
				if !ev.IsCreate() {
					continue
				}
				log.Println("event:", ev)

				ext := strings.ToLower(filepath.Ext(ev.Name))
				if ext == "" {
					continue
				}
				ext = ext[1:]
				i := sort.SearchStrings(movieExtensions, ext)
				if i < len(movieExtensions) && movieExtensions[i] == ext {
					HandleFile(ev.Name, outputPath, *deleteOnCompletion)
				}
			case err := <-watcher.Error:
				log.Println("error:", err)
			}
		}
	}()

	go ui.Run(5000, moviePath, inputPath, movieExtensions, *staticPath)

	// Iterate through and process any existing movie files
	for _, file := range common.FindFileTypes(inputPath, movieExtensions) {
		HandleFile(file, outputPath, *deleteOnCompletion)
	}

	err = watcher.Watch(inputPath)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Watching ", inputPath)
	for {
		time.Sleep(10000 * time.Millisecond)
	}
	watcher.Close()
}
