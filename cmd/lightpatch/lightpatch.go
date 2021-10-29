package main

import (
	"io"
	"log"
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
		Binary     bool
	} `cmd help:"Make a patch file to turn 'before' into 'after'."`

	Apply struct {
		BeforeFile *os.File `arg help:"Before filename"`
		PatchFile  *os.File `arg help:"Patch filename"`
		Binary     bool
	} `cmd help:"Apply a patch file."`
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("error: ")

	ctx := kong.Parse(&CLI)
	switch ctx.Command() {
	case "make <before-file> <after-file>":
		before, err := io.ReadAll(CLI.Make.BeforeFile)
		if err != nil {
			log.Fatalln(err)
		}
		after, err := io.ReadAll(CLI.Make.AfterFile)
		if err != nil {
			log.Fatalln(err)
		}
		var opts []lightpatch.FuncOption
		if CLI.Make.Binary {
			opts = append(opts, lightpatch.WithBinary())
		}

		patch, err := lightpatch.MakePatch(before, after, opts...)
		if err != nil {
			log.Fatalln(err)
		}
		os.Stdout.Write(patch)
	case "apply <before-file> <patch-file>":
		before, err := io.ReadAll(CLI.Apply.BeforeFile)
		if err != nil {
			log.Fatalln(err)
		}
		patch, err := io.ReadAll(CLI.Apply.PatchFile)
		if err != nil {
			log.Fatalln(err)
		}

		var opts []lightpatch.FuncOption
		if CLI.Apply.Binary {
			opts = append(opts, lightpatch.WithBinary())
		}

		after, err := lightpatch.ApplyPatch(before, patch, opts...)
		if err != nil {
			log.Fatalln(err)
		}
		os.Stdout.Write(after)
	default:
		panic(ctx.Command())
	}
}
