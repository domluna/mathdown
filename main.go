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
  -verbose  Verbose output.
  -addr	    HTTP address port. (:8000 default).
`

var (
	help      = flag.Bool("help", false, "show usage")
	addr      = flag.String("addr", ":8082", "http port")
	verbose   = flag.Bool("verbose", false, "verbose output")
	math      = flag.Bool("math", false, "parse LateX expressions")
	homeTempl = template.Must(template.New("home").Parse(homeHTML))
	upgrader  = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	watch *watcher.Watcher
)

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
		"Save a markdown file in the watched directory.",
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
    <style>
* {
    margin: 0;
    padding: 0;
    border: 0;
    font: inherit;
    font-size: 18px;
    vertical-align: baseline;
}

html {
	line-height: 1.2;
	font-family: Verdana, sans-serif;
	background-color: #ededed;
}

h1 {
	font-size: 4rem;
	margin-bottom: 5rem;
	margin-top: .5rem;
	text-align: center;
}

h2 {
	font-size: 2.5rem;
	margin-top: 40px;
	margin-bottom: 8px;
}

h3 {
	font-size: 2.0rem;
	margin-top: 40px;
	margin-bottom: 4px;
}

p {
	margin: 1rem 0;
	line-height: 1.5;
}

p a {
	text-decoration: underline;
	color: #262727;
}

pre {
	font-size: 0.9rem;
	position: relative;
	padding: 1.5rem;
	background-color: #fafafa;
}

pre + pre {
	margin-top: 1px;
}

hr {
  	border: none;
  	height: 100px;
  	width: 150px;
  	margin: 8em auto;
  	background-position: center center;
  	background-repeat: no-repeat;
  	background-size: contain;
}

ol, ul {
	display: block;
}

#container {
	max-width: 750px;
	margin: 0 auto;
	padding-right: 1rem;
	padding-left: 1rem;
}

    </style>
    <script src="//cdnjs.cloudflare.com/ajax/libs/KaTeX/0.1.1/katex.min.js"></script>
    <link rel="stylesheet" href="//cdnjs.cloudflare.com/ajax/libs/KaTeX/0.1.1/katex.min.css">
  </head>
  <body>
    <div id="container">
	<div id="text">{{.Data}}</div>
    </div>
    <script type="text/javascript">
      (function() {
       
function renderKatex(root) { 
  console.log(root);
  var children = root.children;
  for (var i = 0; i < children.length; i++) {
    var node = children[i];
    var inner = node.innerText.trim().split(/\s+/);
    console.log(node, inner);
    if (inner[0] === 'rendermath') {
      // Found a block to run KateX on.
      var expression = inner.slice(1, inner.length).join(' ');
      katex.render(expression, node);
    }
  }
}
 
	var renderMath = {{.Render}}
	var text = document.getElementById("text");
        var conn = new WebSocket("ws://{{.Host}}/ws");
        conn.onclose = function(e) {
	  	console.log('Closing connection')
          	text.innerHTML = "<p>Connection closed</p>";
        }
        conn.onmessage = function(e) {
		console.log('A file updated.');
		text.innerHTML = e.data;
		if (renderMath) {
	    		// use KateX
	    		console.log('Rendering math');
	    		renderKatex(text);
	  	}
	}
	
      })();
    </script>
  </body>
</html>
`
