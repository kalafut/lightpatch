package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/kalafut/jpatch"
)

var flagPatch = flag.Bool("p", false, "apply patch")

func main() {
	flag.Parse()

	if flag.NArg() != 2 {
		flag.Usage()
		return
	}

	f1, err := ioutil.ReadFile(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	f2, err := ioutil.ReadFile(flag.Arg(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	if *flagPatch {
		patched, err := jpatch.ApplyPatch(f1, f2)
		if err != nil {
			fmt.Printf("error applying patch: %s\n", err)
		}
		os.Stdout.Write(patched)
	} else {
		patch := jpatch.MakePatch(f1, f2)
		os.Stdout.Write(patch)
	}

}
