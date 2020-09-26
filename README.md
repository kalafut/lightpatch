# lightpatch  [![Build Status](https://travis-ci.org/kalafut/lightpatch.svg?branch=master)](https://travis-ci.org/kalafut/lightpatch)

lightpatch is a simple library for creating and apply patch files, similar to the functions of `diff` and `patch`. It is available primarily as a Go library, though a simple CLI tool is bundled as well.

The main goals of the project are:

- Byte-level diffs. This means it will work well when line-oriented tools like `diff` won't. An example of this being important would be changes to a single-line minified JSON file. Furthermore, there is no presumuption that the input is in UTF-8 or any other encoding. It's just a stream of bytes.
- The patchfile format is simple and well-defined. A basic decoder can be written in a couple dozen lines.
- Patches may contain an integrity check (CRC-32) which provides strong validation that the output file matches the original input. Note: the CRC is to detect accidental corruption or a patch being applied to the wrong file. It offers little protection against malicious tampering.
- Simple API for developers. The [go-diff](https://github.com/sergi/go-diff) library has over 30 functions, whereas lightpatch has three.

The core diff algorithm is general purpose and is geared towards text. While it will handle binary files fine, it does not have the advanced handling for binary diffs you’d find in specialize tools such as [bsdiff](http://www.daemonology.net/bsdiff/), [xdelta](http://xdelta.org/), etc. On the other hand it is _much_ simpler than these tools, especially for decoding.

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

lightpatch is very fast in the general case, but if you give it two very different files (e.g. two different images), it will try hard to find a diff even when there isn't one. By default it will give up after 5 second (usually plenty of time even for large files), but this is adjustable with the `--t` option.### Go library use

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
| Checksum | K (0x4B) | (Optional) The next 4 bytes are the CRC-32 of _dest_. If present, this much be the final command of the patch file. |

The `len` parameter is [varint encoded](https://developers.google.com/protocol-buffers/docs/encoding#varints). Libraries are readily available to handle this encoding (and even a hand-rolled decoder is only a few lines).

### Checksum (CRC-32)

CRC handling is optional on both ends. An encoder doesn't have to include it, and decoder don't have to verify them. It's better if they do, but in very simple cases the complexity may not be desired. Regardless if a decoder is verifying it or not, it should still return an error if there is data following the CRC, as that is an invalid patch.

The CRC uses the common CRC-32-IEEE polynomial.

### Example

Before:

> For four years I ran this company in Rochester every summer and during
> that time, in partnership with the Shuberts, took over houses in
> Syracuse, Utica, Brooklyn and Philadelphia. They were busy years. In
> Rochester I was company manager, stage director, head
> box-office boy and whenever business got bad I’d write the next week’s
> play to save paying for one. 

After:

> For four years I ran this company in Rochester every summer and during
> that time, in partnership with the Shuberts, took over houses in
> Syracuse, Brooklyn and Philadelphia. They were busy years. In
> Rochester I was company manager, stage director, press agent, head
> box-office boy and whenever business got bad I’d write the next week’s
> play to save paying for one.


Patch (as displayed by `xxd`):
```
00000000: 4392 0144 0743 6349 0d2c 2070 7265 7373  C..D.CcI., press
00000010: 2061 6765 6e74 436f 4b3d eb4d 35          agentCoK=.M5
```

### Credits

lightpatch borrows heavily from:

- [go-diff](https://github.com/sergi/go-diff), which provided the inspiration for this project and is still the source of most of the diff generation code (albeit translated from character to byte handling).
- Google's [Diff-Map-Patch](https://github.com/google/diff-match-patch) library provided the fundamental diff generation algorithms upon which lightpatch, go-diff, and many variants are built upon.
