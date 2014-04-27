package ui

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"fmt"
	"github.com/tangerilli/handbraked/common"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Handler struct {
	staticPath string
}

func (p Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, p.staticPath+"/index.html")
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
	Name     string
	Files    []File
	Children []*Directory
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
		for i := 0; i < len(current.Children); i++ {
			cur := current.Children[i]
			if cur.Name == component {
				target = cur
				break
			}
		}
		if target == nil {
			target = &Directory{Name: component, Files: make([]File, 0), Children: make([]*Directory, 0)}
			current.Children = append(current.Children, target)
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
	sourcePath  string
	watcherPath string
	extensions  []string
}

type QueueItem struct {
	Name string
}

func (p QueueRequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
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

	if r.Method == "GET" {
		var items []QueueItem
		for _, file := range common.FindFileTypes(p.watcherPath, p.extensions) {
			items = append(items, QueueItem{filepath.Base(file)})
		}
		encoder := json.NewEncoder(w)
		encoder.Encode(items)
	}
}

type connection struct {
	// The websocket connection.
	ws *websocket.Conn

	// Buffered channel of outbound messages.
	send chan string
}

func (c *connection) reader() {
	for {
		var message string
		err := websocket.Message.Receive(c.ws, &message)
		if err != nil {
			break
		}
		MessageHub.Broadcast <- message
	}
	c.ws.Close()
}

func (c *connection) writer() {
	for message := range c.send {
		err := websocket.Message.Send(c.ws, message)
		if err != nil {
			break
		}
	}
	c.ws.Close()
}

func statusHandler(ws *websocket.Conn) {
	c := &connection{send: make(chan string, 256), ws: ws}
	MessageHub.register <- c
	defer func() { MessageHub.unregister <- c }()
	go c.writer()
	c.reader()
}

// For status:
//  - Requests to /api/queue should return a JSON list of files in the queue directory
//  - Requests to /api/queue/status should open a websocket connection which sends the
//    filename currently being processed, followed by completion percentages, followed by
//    the next filename and so on
// The webpage should hit /api/queue to create the list of files to be processed (as a collection of QueueItem's)
// It can refresh the collection every 30s or so, or whenever something is queued
// There should be a collection view which renders the collection as a series of progress bars
// A websocket should be setup from /api/queue which finds the appropriate collection element and updates the model progress variable
// which causes a re-render of the progress bar (which implies that each queueitem should have its own subviews)

func Run(port int, filePath string, inputPath string, extensions []string, staticPath string) {
	// start the websocket hub
	go MessageHub.run()

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticPath))))

	handler := Handler{staticPath}
	http.Handle("/", handler)

	sourceHandler := SourceFilesHandler{filePath, extensions}
	http.Handle("/api/files/source", sourceHandler)
	queueHandler := QueueRequestHandler{filePath, inputPath, extensions}
	http.Handle("/api/queue", queueHandler)
	http.Handle("/api/queue/status", websocket.Handler(statusHandler))

	listenAddr := ":" + strconv.FormatInt(int64(port), 10)
	fmt.Println("Starting webserver on ", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
