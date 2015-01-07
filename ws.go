package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"text/template"

	"github.com/domluna/watcher"
	"github.com/gorilla/websocket"
	"github.com/russross/blackfriday"
)

var (
	addr      = flag.String("addr", ":8000", "http port")
	homeTempl = template.Must(template.New("MD").Parse(homeHTML))
	upgrader  = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	watch *watcher.Watcher
)

// readFile reads the file from the given path.
func readFile(path string) ([]byte, error) {
	p, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// writer writes the message back to the client
func writer(ws *websocket.Conn, w *watcher.Watcher) {
	fchan := w.Watch()
	defer func() {
		ws.Close()
		w.Close()
	}()

	for {
		select {
		case fi, ok := <-fchan:
			if !ok {
				break
			}

			if fi.Ext != ".md" {
				continue

			}

			log.Println(fi)
			p, err := readFile(fi.Path)

			p = blackfriday.MarkdownBasic(p)
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
	log.Println("Setting up websockets")
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return

	}

	go writer(ws, watch)
	reader(ws)
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var v = struct {
		Host string
	}{
		r.Host,
	}
	homeTempl.Execute(w, &v)
}

func main() {
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

	log.Println("Starting up MathDown ...")
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal(err)
	}

}

const homeHTML = `
<!DOCTYPE html>
<html lang="en">
  <head>
    <title> MathDown </title>
  </head>
  <body>
    <h1> MathDown </h1>
    <pre id="text"></pre> 
    <script type="text/javascript">
      (function() {
      	    var data = document.getElementById("text");
              var conn = new WebSocket("ws://{{.Host}}/ws");
              conn.onclose = function(e) {
                data.textContent = "Connection closed";
              }
              conn.onmessage = function(e) {
              	    console.log("file updated");
          	    data.innerText = e.data;
              }
      })();
    </script>
  </body>
</html>
`
