package transpiler

import (
	"fmt"
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
	p := parser.New(string(data))
	for n := p.Next(); n != nil; n = p.Next() {
		fmt.Fprintln(stdout, n.HTML())
	}
	if warns := p.Warnings(); len(warns) > 0 {
		for _, w := range warns {
			fmt.Fprintln(stderr, w)
		}
	}
	return nil
}
