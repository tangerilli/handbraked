package main

import ("fmt"
        "flag"
        "os"
        "path/filepath"
        "os/exec"
)

func usage() {
    fmt.Println("usage: handbraked [path to watch]")
    flag.PrintDefaults()
    os.Exit(1)
}

func findMovies(path string) []string {
    var results []string
    formats := []string{"m4v", "avi", "mpg", "mpeg", "mkv"}
    for _, pattern := range formats {
        r, _ := filepath.Glob(filepath.Join(path, "*." + pattern))
        results = append(results, r...)
    }
    return results
}

func processFile(path string, c chan string) {
    defer close(c)

    fmt.Println("Doing stuff with " + path)
    cmd := exec.Command("/usr/bin/file", path)
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        fmt.Println("Error opening stdout")
        return
    }

    go func() {
        b := make([]byte, 1024)
        for l, err := stdout.Read(b); l > 0 || err == nil; l, err = stdout.Read(b) {
            if l > 0 {
                c <- string(b)
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

func main() {
    flag.Usage = usage
    flag.Parse()
    args := flag.Args()
    if len(args) < 1 {
        fmt.Println("Path is missing")
        os.Exit(1)
    }

    files := findMovies(args[0])
    for _, file := range files {
        c := make(chan string)
        go processFile(file, c)
        for output := range c {
            fmt.Println(output)
        }
        fmt.Println("Finished " + file)
    }
}