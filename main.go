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

	"github.com/domluna/watcher"
	"github.com/gorilla/websocket"
	"github.com/russross/blackfriday"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: mdpreview [options]\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Hot renders markdown files in current directory and sub directories on save in the browser.\n")
	fmt.Fprintf(os.Stderr, "\n")
	flag.PrintDefaults()
	os.Exit(2)
}

var (
	addr     = flag.Int("addr", 8000, "http port")
	verbose  = flag.Bool("verbose", false, "verbose output, for debug purposes mainly")
	tmpl     = template.Must(template.New("home").Parse(homeHTML))
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	watch *watcher.Watcher
)

func main() {
	flag.Usage = usage
	flag.Parse()

	fmt.Println(*verbose)
	fmt.Println(*addr)

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	// watch for changes to markdown files
	watch, err = watcher.New(wd, ".md")
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", handlerPreview)
	http.HandleFunc("/ws", handlerWS)

	hostURL := fmt.Sprintf("http://localhost:%d", *addr)
	//browser.OpenURL(hostURL)
	debug("Starting up Markdown Preview at " + hostURL)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", *addr), nil); err != nil {
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

			// skip all non-md signals
			if fi.Ext != ".md" {
				continue

			}

			debug("FileEvent: %s", fi)

			p, err := readFile(fi.Path)
			p = blackfriday.MarkdownCommon(p)
			debug("%s", p)

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

const homeHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
<title> Markdown Preview </title>
</head>
<body>
	<div id="container">
		<div id="text">{{.Data}}</div>
	</div>
	<script type="text/javascript">
		(function() {
			var data = document.getElementById("text");
			var conn = new WebSocket("ws://{{.Host}}/ws");
			conn.onclose = function(e) {
				data.textContent = 'Connection closed';
			}
			conn.onmessage = function(e) {
				console.log('file updated');
				text.innerHTML = e.data;
			}
		})();
	</script>
</body>
</html>
`
