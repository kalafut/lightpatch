# lightpatch  [![Build Status](https://travis-ci.org/kalafut/lightpatch.svg?branch=master)](https://travis-ci.org/kalafut/lightpatch)

lightpatch is a simple library for creating and apply patch files, similar to the functions of `diff` and `patch`. It is available primarily as a Go library, though a simple CLI tool is bundled as well.

The main goals of the project are:

- Efficient patching of files. The patch files themselves are not intended to be human-readable diffs.
- Byte-level diffs. This means it will work well when line-oriented tools like `diff` won't. An example of this being important would be changes to a single-line minified Javascript file. Furthermore, there is no presumuption that the input is in UTF-8 or any other encoding. It's just a stream of bytes.
- The patchfile format is simple and well-defined. A basic decoder can be written in a few dozen lines. (Contrast to [VCDIFF](https://en.wikipedia.org/wiki/VCDIFF) and its [RFC](https://tools.ietf.org/html/rfc3284) that is nearly 30 pages long.)
- Patches may contain an integrity check (CRC-32) which provides strong validation that the output file matches the original input. Note: the CRC is to detect accidental corruption or a patch being applied to the wrong file. It offers little protection against malicious tampering.
- Simple API for developers. The [go-diff](https://github.com/sergi/go-diff) library has over 30 functions, whereas lightpatch has three.

The core diff algorithm is general purpose and is geared towards text. While it will handle binary files fine (e.g. you could efficiently patch some changed EXIF data in a JPG image), it does not have the advanced handling of binary diffs you'll find in specialized tools such as [bsdiff](http://www.daemonology.net/bsdiff/), [xdelta](http://xdelta.org/), etc. On the other hand it is _much_ simpler than those tools, especially for decoding.

### CLI use

You can easily build from source by installing [Go](https://golang.org) and running:

```
go get -u github.com/kalafut/lightpatch/cmd/lightpatch
```

Making and applying patch files are the two supported commands:

```
lightpatch make file1 file2 > patch
lightpatch apply file1 patch > output        # should match file2
lightpatch make --t 30s file1 file2 > patch  # allow 30s to make the patch
```

lightpatch is very fast in the general case, but if you give it two very different files, it will try hard to find a diff even when there isn't one. By default it will "give up" after 5 seconds (usually plenty of time even for large files), but this is adjustable with the `--t` option. 

Note: the command still succeeds even if the timeout is reached, but the output might be a naïve diff that is just the new file in its entirety.

### Go library use

The API is described in the [docs](https://pkg.go.dev/github.com/kalafut/lightpatch). The [source for the CLI tool](https://github.com/kalafut/lightpatch/blob/master/cmd/lightpatch/lightpatch.go) is also a good example.

### File Format

The lightpatch file format is a simple [TLV](https://en.wikipedia.org/wiki/Type-length-value) style. The patch file provide edit instruction to be applied to a source file. The command format is:

`command [len] [data]`

### Commands

| Command  | Code | Notes           |
| -------- | ---- | --------------- |
| Copy     | C (0x43) | Copy `len` bytes from _source_ to _dest_. `data` is not used.| 
| Insert   | I (0x49) | Insert the next `len` bytes from `data` into _dest_. |
| Delete   | D (0x44) | "Delete" the next `len` _source_ bytes by advancing the source input and output nothing to _dest_. `data` is not used. | 
| Checksum | K (0x4B) | (Optional) The next 4 bytes are the CRC-32 of _dest_. If present, this must be the final command of the patch file. |

The `len` parameter is [varint encoded](https://developers.google.com/protocol-buffers/docs/encoding#varints). Libraries are readily available to handle this encoding (and even a hand-rolled decoder is only a few lines).

### Checksum (CRC-32)

CRC handling is optional on both ends. An encoder doesn't have to include it, and decoder don't have to verify them. It's better if they do, but in very simple cases the complexity may not be desired. Regardless if a decoder is verifying it or not, it should still return an error if there is data following the CRC, as that is an invalid patch.

The CRC uses the common CRC-32-IEEE polynomial.

### Example

Before:

`The quick brown fox jumped over the lazy dog`

After:

`The quick brown fox leaped over the lazy dog.`

Patch (hex):

```
43 14          copy 20 bytes ("The quick brown fox ")
44 03          delete/skip 3 bytes ("jum")
49 03 6c 65 61 insert 3 bytes ("lea")
43 15          copy 21 bytes ("ped over the lazy dog")
49 01 2e       insert 1 byte (".")
4b 96 f6 b7 6c CRC-32 is 0x96f6b76c
```

### Credits

lightpatch borrows heavily from:

- [go-diff](https://github.com/sergi/go-diff), which provided the inspiration for this project and is still the source of most of the diff generation code (albeit translated from character to byte handling).
- Google's [Diff-Map-Patch](https://github.com/google/diff-match-patch) library provided the fundamental diff generation algorithms upon which lightpatch, go-diff, and many variants are built upon.
