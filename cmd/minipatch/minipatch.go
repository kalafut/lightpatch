package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/kalafut/minipatch"
)

var CLI struct {
	Make struct {
		BaseFile   *os.File      `arg help:"Base filename"`
		UpdateFile *os.File      `arg help:"Updated filename"`
		TimeLimit  time.Duration `name:"t" help:"Max time to build patch."`
	} `cmd help:"Make a patch file to turn 'base' into 'update'."`

	Apply struct {
		BaseFile  *os.File `arg help:"Base filename"`
		PatchFile *os.File `arg help:"Patch filename"`
	} `cmd help:"Apply a patch file."`
}

func mustReadAll(f *os.File) []byte {
	d, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return d
}

func main() {
	ctx := kong.Parse(&CLI)
	switch ctx.Command() {
	case "make <base-file> <update-file>":
		patch := minipatch.MakePatch(
			mustReadAll(CLI.Make.BaseFile),
			mustReadAll(CLI.Make.UpdateFile),
		)
		os.Stdout.Write(patch)
	case "apply <base-file> <patch-file>":
		patched, err := minipatch.ApplyPatch(
			mustReadAll(CLI.Apply.BaseFile),
			mustReadAll(CLI.Apply.PatchFile),
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error applying patch: %s\n", err)
			os.Exit(1)
		}
		os.Stdout.Write(patched)
	default:
		panic(ctx.Command())
	}
}
