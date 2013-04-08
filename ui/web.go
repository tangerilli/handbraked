package ui

import ("net/http"
        "strconv"
        "fmt"
        "log"
        "os"
        "path/filepath"
        "strings"
        "sort"
        "encoding/json"
)

type Handler struct {
}

func (p Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "static/index.html")
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
    path := r.URL.String()[1:]
    http.ServeFile(w, r, path)
}
type SourceFilesHandler struct {
    sourcePath string
    extensions []string
}

type File struct {
    Name string
    Path string
}

type Directory struct {
    Name string
    Files []File
    Children []Directory
}

func getDirectory(path string, root *Directory) *Directory {
    // Unix only for now
    dirpath, _ := filepath.Split(path)
    components := strings.Split(dirpath, "/")
    if len(components) == 1 {
        return root
    }

    current := root
    for _, component := range components[1:] {
        if component == "" {
            continue
        }
        var target *Directory = nil
        for i:=0; i<len(current.Children); i++ {
            cur := &current.Children[i]
            if cur.Name == component {
                target = cur
                break
            }
        }
        if target == nil {
            target = &Directory{Name:component, Files:make([]File, 0), Children:make([]Directory, 0)}
            current.Children = append(current.Children, *target)
        }
        current = target
    }
    return current
}

func (p SourceFilesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    var root Directory
    root.Name = "/"
    filepath.Walk(p.sourcePath, func(path string, info os.FileInfo, err error) error {
        localPath := strings.Replace(path, p.sourcePath, "", -1)
        dir := getDirectory(localPath, &root)

        ext := strings.ToLower(filepath.Ext(path))
        if ext == "" {
            return nil
        }
        // Get rid of the leading '.'
        ext = ext[1:]
        i := sort.SearchStrings(p.extensions, ext)
        if i >= len(p.extensions) || p.extensions[i] != ext {
            return nil
        }
        f := File{filepath.Base(localPath), localPath}
        dir.Files = append(dir.Files, f)
        return nil
    })
    encoder := json.NewEncoder(w)
    encoder.Encode(root)
}

type queueRequest struct {
    Path string
}

type QueueRequestHandler struct {
    sourcePath string
    watcherPath string
}

func (p QueueRequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    decoder := json.NewDecoder(r.Body)
    var request queueRequest
    err := decoder.Decode(&request)
    if err != nil {
        fmt.Println(err)
        panic(1)
    }
    path := filepath.Join(p.sourcePath, request.Path)
    fmt.Println("Need to symlink " + path + " to " + p.watcherPath)
    err = os.Symlink(path, filepath.Join(p.watcherPath, filepath.Base(path)))
    if err != nil {
        fmt.Println("Error!")
    }
}

func Run(port int, filePath string, inputPath string, extensions []string) {
    handler := Handler{}
    http.Handle("/", handler)
    http.HandleFunc("/static/", staticHandler)
    sourceHandler := SourceFilesHandler{filePath, extensions}
    http.Handle("/api/files/source", sourceHandler)
    queueHandler := QueueRequestHandler{filePath, inputPath}
    http.Handle("/api/files/queue", queueHandler)

    listenAddr := ":" + strconv.FormatInt(int64(port), 10)
    fmt.Println("Starting webserver on ", listenAddr)
    log.Fatal(http.ListenAndServe(listenAddr, nil))
}