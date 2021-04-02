package ui

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/thechriswalker/go-astris/astris"
)

var indexTemplate = template.Must(template.New("index").Parse(
	`<!DOCTYPE html>
<html>
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
	<meta http-equiv="Content-Security-Policy" content="default-src 'self'; script-src 'self' '{{.MetaIntegrity}}'" />
    <title>Astris • {{ .Title }}</title>
    <link rel="stylesheet" href="/assets/bulma.min.css" />
	<link rel="shortcut icon" type="image/png" href="/assets/astris-logo-128.png" />
  </head>
  <body>
    <noscript>
      <section class="hero is-dark">
        <div class="hero-body">
          <h1 class="title">Astris</h1>
          <p class="subtitle">Astris requires JavaScript to function, please enable and reload the page.</p>
        </div>
      </section>
    </noscript>
    <div id="app"></div>
	<script type="text/javascript">{{.MetaScript}}</script>
    <script type="module" src="{{ .JSFile }}"></script>
  </body>
</html>
`,
))

var assetsHTTP = http.FileServer(http.FS(Assets))
var builtHTTP = http.FileServer(http.FS(Built))

// I'll keep this map here to keep the UI consistent.
type page struct {
	Title  string
	JSFile string
}

var (
	metaJS   template.JS
	metaHash template.HTMLAttr
)

var metaOnce sync.Once

func generateMeta() {
	b, err := json.Marshal(map[string]string{
		"version":   astris.Version,
		"commit":    astris.Commit,
		"buildDate": astris.BuildDate,
	})
	// ignore the error, it should not happen,
	// as we know ths input type. only a memory
	// allocation fail could bail this.
	if err != nil {
		panic(err)
	}
	// now our JS is `window.META=...`
	str := fmt.Sprintf("window.META=%s;", b)
	h := sha256.New()
	h.Write([]byte(str))
	metaHash = template.HTMLAttr("sha256-" + base64.StdEncoding.EncodeToString(h.Sum(nil)))
	metaJS = template.JS(str)
}

func (p *page) MetaScript() template.JS {
	metaOnce.Do(generateMeta)
	return metaJS
}

func (p *page) MetaIntegrity() template.HTMLAttr {
	metaOnce.Do(generateMeta)
	return metaHash
}

func (p *page) Mux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/assets/", assetsHTTP)
	mux.Handle(p.JSFile, builtHTTP)
	mux.Handle(p.JSFile+".map", builtHTTP)
	mux.HandleFunc("/authority/", func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(200)
		err := indexTemplate.Execute(rw, p)
		if err != nil {
			log.Error().Err(err).Str("page", p.Title).Msg("Error rendering index for page")
		}
	})
	return mux
}

var AuthorityPage = &page{
	Title:  "Election Authority",
	JSFile: "/authority/index.js",
}
