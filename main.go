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
	"github.com/pkg/browser"
	"github.com/russross/blackfriday"
)

const usage = `MathDown 

Converts markdown to html to be displayed in the browser one edit.

Options:
  -help     Show this screen.
  -math	    Parse LateX expressions (currently not working)
  -debug    Verbose debug output.
  -addr	    HTTP address port. (:8000 default).
`

var (
	help      = flag.Bool("help", false, "show usage")
	addr      = flag.String("addr", ":8000", "http port")
	verbose   = flag.Bool("debug", false, "verbose output")
	math      = flag.Bool("math", false, "parse LateX expressions")
	homeTempl = template.Must(template.New("home").Parse(homeHTML))
	upgrader  = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	watch *watcher.Watcher
)

// debug is a utility function to print out
// logs. Activated based on the debug flag.
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

// writer writes the message back to the client.
func writer(ws *websocket.Conn) {
	watch.Watch()
	defer func() {
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

			if *math {
				p = blackfriday.Markdown(p, blackfriday.LatexRenderer(0), 0)
			} else {
				p = blackfriday.MarkdownCommon(p)
			}
			debug("%s", p)

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

// serveWS sets up the websocket connection.
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

// serveHome serves the initial template.
func serveHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var v = struct {
		Host   string
		Data   string
		Render bool
	}{
		r.Host,
		"Save a file to parse in realtime.",
		false,
	}

	if *math {
		v.Render = true
	}

	homeTempl.Execute(w, &v)
}

func main() {
	flag.Parse()

	if *help {
		fmt.Print(usage)
		os.Exit(0)
	}

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

	browser.OpenURL(fmt.Sprintf("http://localhost%s", *addr))

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
    <script src="//cdnjs.cloudflare.com/ajax/libs/KaTeX/0.1.1/katex.min.js"></script>
    <link rel="stylesheet" href="//cdnjs.cloudflare.com/ajax/libs/KaTeX/0.1.1/katex.min.css">
  </head>
  <body>
    <pre id="text">{{.Data}}</pre> 
    <script type="text/javascript">
      (function() {
	var renderMath = {{.Render}}
	var text = document.getElementById("text");
        var conn = new WebSocket("ws://{{.Host}}/ws");
        conn.onclose = function(e) {
	  console.log('Closing connection')
          data.innerHTML = "<p>Connection closed</p>";
        }
        conn.onmessage = function(e) {
	  console.log('A file updated.')
	  if (renderMath) {
	    // use KateX
	    katex.render(e.data, text)
	  } else {
            text.innerHTML = e.data;
	  }
        }
      })();
    </script>
  </body>
</html>
`
