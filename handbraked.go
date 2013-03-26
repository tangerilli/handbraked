package main

import ("fmt"
        "flag"
        "os"
        "path/filepath"
        "os/exec"
        "regexp"
        "strconv"
        "github.com/howeyc/fsnotify"
        "log"
        "time"
        "sort"
        "strings"
)

var movieExtensions []string = []string{"m4v", "avi", "mpg", "mpeg", "mkv", "mov"}

func FindFileTypes(path string, extensions []string) []string {
    var results []string
    for _, pattern := range extensions {
        r, err := filepath.Glob(filepath.Join(path, "*." + pattern))
        if err != nil {
            continue
        }
        results = append(results, r...)
    }
    return results
}

func usage() {
    fmt.Println("usage: handbraked [path to watch] [output path]")
    flag.PrintDefaults()
    os.Exit(1)
}

func processFile(path string, outputPath string, c chan float64) {
    defer close(c)

    fmt.Println("Processing " + path)
    outputFile := filepath.Join(outputPath, filepath.Base(path))
    outputFile = outputFile[:len(outputFile)-3] + "m4v"
    cmd := exec.Command("HandBrakeCLI", "-i", path, "--preset=iPad", "-o", outputFile)
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
}

func HandleFile(file string, outputPath string, deleteOnCompletion bool) {
    c := make(chan float64)
    go processFile(file, outputPath, c)
    for output := range c {
        fmt.Println(output)
    }
    fmt.Println("Finished " + file)
    if deleteOnCompletion {
        os.Remove(file)
    }
}

func main() {
    flag.Usage = usage
    var deleteOnCompletion = flag.Bool("delete", true, "Delete files after processing")
    flag.Parse()
    args := flag.Args()
    if len(args) < 2 {
        usage()
    }
    inputPath := args[0]
    outputPath := args[1]

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
                log.Println("event:", ev)
                if !ev.IsCreate() {
                    continue
                }
                log.Println("event:", ev)

                ext := strings.ToLower(filepath.Ext(ev.Name)[1:])
                i := sort.SearchStrings(movieExtensions, ext)
                if i < len(movieExtensions) && movieExtensions[i] == ext {
                    HandleFile(ev.Name, outputPath, *deleteOnCompletion)
                }
            case err := <-watcher.Error:
                log.Println("error:", err)
            }
        }
    }()

    // Iterate through and process any existing movie files
    for _, file := range FindFileTypes(inputPath, movieExtensions) {
        HandleFile(file, outputPath, *deleteOnCompletion)
    }

    err = watcher.Watch(inputPath)
    if err != nil {
        log.Fatal(err)
    }
    log.Println("Watching ", args[0])
    for {
        time.Sleep(10 * time.Second)
    }

    watcher.Close()
}