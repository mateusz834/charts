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
	// Parse and execute templates, they do not change over time.
	indexContent    = mustParseAndExec("tmpls/layout.html", "tmpls/index.html")
	mySharesContent = mustParseAndExec("tmpls/layout.html", "tmpls/my-shares.html")
	shareContent    = mustParseAndExec("tmpls/layout.html", "tmpls/share.html")
)

func mustParseAndExec(templates ...string) []byte {
	index := template.Must(template.ParseFS(tmpls, templates...))
	var buf bytes.Buffer
	if err := index.Execute(&buf, nil); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func Index(w io.Writer) error {
	_, err := w.Write(indexContent)
	return err
}

func MyShares(w io.Writer) error {
	_, err := w.Write(mySharesContent)
	return err
}

func Share(w io.Writer) error {
	_, err := w.Write(shareContent)
	return err
}
