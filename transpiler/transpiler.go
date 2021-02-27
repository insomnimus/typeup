package transpiler

import (
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"typeup/parser"
)

func ToHTML(stdin io.Reader, stdout, stderr io.Writer) error {
	// NOTE: maybe use io.ReadAll()? but for compatibility with older go versions, idk
	data, err := ioutil.ReadAll(stdin)
	if err != nil {
		return err
	}

	var content []string
	p := parser.New(string(data))
	for n := p.Next(); n != nil; n = p.Next() {
		content = append(content, n.HTML())
	}
	if meta := p.Metas(); len(meta) > 0 {
		for key, val := range meta {
			fmt.Fprintln(stderr, key, "=", val)
		}
	}
	if warns := p.Warnings(); len(warns) > 0 {
		for _, w := range warns {
			fmt.Fprintln(stderr, w)
		}
	}
	doc := "<html>"
	if title, ok := p.Meta("title"); ok {
		doc += fmt.Sprintf("\n<head> <title>\n %s \n</title> </head>", html.EscapeString(title))
	}
	doc += "\n<body>"
	fmt.Fprintln(stdout, doc)
	for _, x := range content {
		fmt.Fprintln(stdout, x)
	}
	fmt.Fprint(stdout, "</body>\n</html>")

	return nil
}
