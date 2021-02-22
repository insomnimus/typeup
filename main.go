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
		err error
	)
	switch flag.NArg() {
	case 0:
		in = os.Stdin
		out = os.Stdout
	case 1:
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			log.Fatal(err)
		}
		in = f
		defer f.Close()
		out = os.Stdout
	case 2:
		fi, err := os.Open(flag.Arg(0))
		if err != nil {
			log.Fatal(err)
		}
		defer fi.Close()
		in = fi
		fo, err := os.Create(flag.Arg(1))
		if err != nil {
			log.Fatal(err)
		}
		out = fo
		defer fo.Close()
	}
	err = transpiler.ToHTML(in, out, os.Stderr)
	if err != nil {
		log.Fatal(err)
	}
}
