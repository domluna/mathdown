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

const usage = `
`

var (
	addr      = flag.String("addr", ":8000", "http port")
	verbose   = flag.Bool("v", false, "verbose debug output")
	homeTempl = template.Must(template.New("home").Parse(homeHTML))
	upgrader  = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	watch *watcher.Watcher
)

func debug(msg string, args ...interface{}) {
	if *verbose {
		if len(args) > 0 {
			log.Println(fmt.Sprintf(msg, args))
		} else {
			log.Println(msg)
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

// writer writes the message back to the client
func writer(ws *websocket.Conn) {
	fchan := watch.Watch()
	defer func() {
		ws.Close()
		watch.Close()
	}()

	for {
		select {
		case fi, ok := <-fchan:
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
			if err = ws.WriteMessage(websocket.TextMessage, p); err != nil {
				break
			}
		}
	}
}

func reader(ws *websocket.Conn) {
	defer ws.Close()
	ws.SetReadLimit(512)

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}
}

func serveWS(w http.ResponseWriter, r *http.Request) {
	debug("Setting up websockets")
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return

	}

	go writer(ws)
	reader(ws)
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var v = struct {
		Host string
		Data string
	}{
		r.Host,
		"Save a file to parse in realtime.",
	}
	homeTempl.Execute(w, &v)
}

func main() {
	flag.Parse()
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	watch, err = watcher.New(wd, ".md")
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", serveWS)

	debug("Starting up MathDown on port %s", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal(err)
	}

}

const homeHTML = `
<!DOCTYPE html>
<html lang="en">
  <head>
    <title>MathDown</title>
  </head>
  <body>
    <h1> MathDown </h1>
    <pre id="text">{{.Data}}</pre> 
    <script type="text/javascript">
      (function() {
	var data = document.getElementById("text");
        var conn = new WebSocket("ws://{{.Host}}/ws");
        conn.onclose = function(e) {
        	data.innerHTML = "<p>Connection closed</p>";
        }
        conn.onmessage = function(e) {
              	console.log("file updated");
          	data.innerHTML = e.data;
        }
      })();
    </script>
  </body>
</html>
`
