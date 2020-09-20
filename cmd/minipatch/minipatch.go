package main

import (
	"fmt"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/kalafut/minipatch"
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
		if err := minipatch.MakePatchTimeout(
			CLI.Make.BeforeFile,
			CLI.Make.AfterFile,
			os.Stdout,
			CLI.Make.TimeLimit,
		); err != nil {
			fmt.Fprintf(os.Stderr, "error creating patch: %s\n", err)
			os.Exit(1)
		}
	case "apply <before-file> <patch-file>":
		if err := minipatch.ApplyPatch(
			CLI.Apply.BeforeFile,
			CLI.Apply.PatchFile,
			os.Stdout,
		); err != nil {
			fmt.Fprintf(os.Stderr, "error applying patch: %s\n", err)
			os.Exit(1)
		}
	default:
		panic(ctx.Command())
	}
}
