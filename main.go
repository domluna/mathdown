// TODO: use go-bindata for template/css
// TODO: better css
package main

import (
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/domluna/watcher"
	"github.com/gorilla/websocket"
	"github.com/russross/blackfriday"
)

var (
	port     = flag.Int("port", 8000, "http port")
	verbose  = flag.Bool("verbose", false, "verbose output, for debug purposes mainly")
	tmpl     = template.Must(template.ParseFiles(path.Join("templates", "preview.html")))
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	watch *watcher.Watcher
)

func main() {
	flag.Usage = usage
	flag.Parse()

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// watch for changes to markdown files
	watch, err = watcher.New(wd, []string{"md"})
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", handlerPreview)
	http.HandleFunc("/ws", handlerWS)

	hostURL := fmt.Sprintf("http://localhost:%d", *port)
	fmt.Println("Starting up Markdown Preview at " + hostURL)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); err != nil {
		log.Fatal(err)
	}

}

// debug is a utility function to print out
// logs. Activated based on the verbose flag.
func debug(msg string, args ...interface{}) {
	if *verbose {
		if len(args) > 0 {
			log.Printf(msg, args...)
		} else {
			log.Printf(msg)
		}
	}
}

// readFile reads the file from the given path.
func readFile(path string) ([]byte, error) {
	p, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// writer writes the message back to the client.
func writer(ws *websocket.Conn) {
	watch.Watch()

	defer func() {
		debug("Shutting down writer")
		ws.Close()
		watch.Close()
	}()

	for {
		select {
		case fi, ok := <-watch.Events:

			if !ok {
				break
			}

			debug("FileEvent: %s", fi)

			p, err := readFile(fi.Path)
			p = blackfriday.MarkdownCommon(p)

			if err = ws.WriteMessage(websocket.TextMessage, p); err != nil {
				break
			}
		}
	}
}

func reader(ws *websocket.Conn) {
	defer func() {
		debug("Shutting down reader")
		ws.Close()
	}()
	ws.SetReadLimit(512)

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}
}

// handlerWS sets up the websocket connection.
func handlerWS(w http.ResponseWriter, r *http.Request) {
	debug("Setting up websockets")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	go writer(conn)
	reader(conn)
}

// handlerPreview serves the initial template.
func handlerPreview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var v = struct {
		Host string
		Data string
	}{
		r.Host,
		"Go ahead, save that markdown file.",
	}

	tmpl.Execute(w, &v)
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: mdpreview [options]\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Hot renders markdown files in current directory and sub directories on save in the browser.\n")
	fmt.Fprintf(os.Stderr, "\n")
	flag.PrintDefaults()
	os.Exit(2)
}
