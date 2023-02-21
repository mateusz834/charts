package templates

import (
	"bytes"
	"embed"
	"html/template"
	"io"
)

//go:embed tmpls
var tmpls embed.FS

var (
	indexContent []byte
)

func init() {
	// Parse and execute the index template, it does not change over time.
	index := template.Must(template.ParseFS(tmpls, "tmpls/layout.html", "tmpls/index.html"))
	var buf bytes.Buffer
	if err := index.Execute(&buf, nil); err != nil {
		panic(err)
	}
	indexContent = buf.Bytes()
}

func Index(w io.Writer) error {
	_, err := w.Write(indexContent)
	return err
}
