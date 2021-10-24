package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/kalafut/lightpatch"
)

var CLI struct {
	Make struct {
		BeforeFile *os.File      `arg help:"Before file"`
		AfterFile  *os.File      `arg help:"After file"`
		TimeLimit  time.Duration `name:"t" default:"5s" help:"Max time to build patch."`
	} `cmd help:"Make a patch file to turn 'before' into 'after'."`

	Apply struct {
		BeforeFile *os.File `arg help:"Before filename"`
		PatchFile  *os.File `arg help:"Patch filename"`
	} `cmd help:"Apply a patch file."`
}

func main() {
	ctx := kong.Parse(&CLI)
	switch ctx.Command() {
	case "make <before-file> <after-file>":
		before, err := io.ReadAll(CLI.Make.BeforeFile)
		if err != nil {
			panic(err)
		}
		after, err := io.ReadAll(CLI.Make.AfterFile)
		if err != nil {
			panic(err)
		}
		patch := lightpatch.MakePatch(before, after)
		os.Stdout.Write(patch)
	case "apply <before-file> <patch-file>":
		before, err := io.ReadAll(CLI.Apply.BeforeFile)
		if err != nil {
			panic(err)
		}
		patch, err := io.ReadAll(CLI.Apply.PatchFile)
		if err != nil {
			panic(err)
		}
		after, err := lightpatch.ApplyPatch(before, patch)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nerror applying patch: %s\n", err)
			os.Exit(1)
		}
		os.Stdout.Write(after)
	default:
		panic(ctx.Command())
	}
}
