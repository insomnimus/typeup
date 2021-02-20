package main

import (
	"flag"
	"io"
	"log"
	"os"
	"typeup/transpiler"
)

func main() {
	log.SetFlags(0)
	flag.Parse()
	var (
		in  io.Reader
		out io.Writer
	)
	switch flag.NArg() {
	case 0:
		in = os.Stdin
		out = os.Stdout
	case 1:
		in, err := os.Open(flag.Arg(0))
		if err != nil {
			log.Fatal(err)
		}
		defer in.Close()
	case 2:
		in, err := os.Open(flag.Arg(0))
		if err != nil {
			log.Fatal(err)
		}
		out, err := os.Open(flag.Arg(1))
		if err != nil {
			log.Fatal(err)
		}
		defer in.Close()
		defer out.Close()
	}
	err := transpiler.ToHTML(in, out, os.Stderr)
	if err != nil {
		log.Fatal(err)
	}
}
