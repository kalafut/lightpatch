// Package lightpatch generates and applies patch files. A description of the patch file
// format is included in the README.
package lightpatch

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"math"
	"strconv"
	"unicode/utf8"

	"github.com/sergi/go-diff/diffmatchpatch"
)

const (
	Version  = 'A'
	OpCopy   = 'C'
	OpInsert = 'I'
	OpDelete = 'D'
	OpCRC    = 'K'
)

var (
	opmap = map[diffmatchpatch.Operation]byte{
		diffmatchpatch.DiffEqual:  OpCopy,
		diffmatchpatch.DiffInsert: OpInsert,
		diffmatchpatch.DiffDelete: OpDelete,
	}

	ErrCRC = errors.New("CRC mismatch")
)

// MakePatch generates a diff to change before into after, writing the output to patch.
func MakePatch(before, after []byte, o ...FuncOption) ([]byte, error) {
	var cfg config
	var patch []byte

	for _, f := range o {
		f(&cfg)
	}

	var beforeStr string
	var afterStr string
	if cfg.binary {
		beforeStr = hex.EncodeToString(before)
		afterStr = hex.EncodeToString(after)
	} else {
		if !utf8.Valid(before) {
			return nil, errors.New("non-utf8 data in 'before' data")
		}
		if !utf8.Valid(after) {
			return nil, errors.New("non-utf8 data in 'after' data")
		}

		beforeStr = string(before)
		afterStr = string(after)
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(beforeStr, afterStr, false)

	// If inputs are very different, the total size of the encoded diffs can be greater than just
	// outputting after bytes. Check whether this "naive" diff is actually shorter.
	if len(afterStr) < encodedLen(diffs) {
		diffs = []diffmatchpatch.Diff{
			{
				Type: diffmatchpatch.DiffInsert,
				Text: string(afterStr),
			},
		}
	}

	patch = append(patch, Version)

	for _, diff := range diffs {
		patch = append(patch, []byte(fmt.Sprintf("%x", len(diff.Text)))...)
		patch = append(patch, []byte{opmap[diff.Type]}...)

		if diff.Type == diffmatchpatch.DiffInsert {
			patch = append(patch, []byte(diff.Text)...)
		}
	}

	var crc uint32
	if !cfg.noCRC {
		crc = crc32.ChecksumIEEE(after)
	}
	patch = append(patch, []byte(fmt.Sprintf("%x%c", crc, OpCRC))...)

	return patch, nil
}

// ApplyPatch reads before, applies the edits from patch, and writes
// the output to after.
func ApplyPatch(beforeByte, patchByte []byte, o ...FuncOption) ([]byte, error) {
	var cfg config

	for _, f := range o {
		f(&cfg)
	}

	if cfg.binary {
		beforeByte = []byte(hex.EncodeToString(beforeByte))
	}

	after := new(bytes.Buffer)
	beforeBR := bufio.NewReader(bytes.NewReader(beforeByte))
	patchBR := newTrackedReader(patchByte)

	ver, err := patchBR.ReadByte()
	if err != nil {
		return nil, err
	}

	if ver != Version {
		return nil, fmt.Errorf("unknown version %q", ver)
	}

	for {
		tl, op, err := readOp(patchBR)
		if err == io.EOF {
			return nil, io.ErrUnexpectedEOF
		} else if err != nil {
			return nil, err
		}

		switch op {
		case OpCopy:
			_, err = io.CopyN(after, beforeBR, int64(tl))
		case OpInsert:
			_, err = io.CopyN(after, patchBR, int64(tl))
		case OpDelete:
			_, err = beforeBR.Discard(tl)
		case OpCRC:
			all := after.Bytes()
			if cfg.binary {
				dst := make([]byte, hex.DecodedLen(len(all)))
				_, err = hex.Decode(dst, all)
				if err != nil {
					return nil, err
				}
				all = dst
			}
			crc := uint32(tl)
			if !cfg.noCRC && crc != 0 && crc32.ChecksumIEEE(all) != crc {
				return nil, ErrCRC
			}

			return all, nil

		default:
			return nil, fmt.Errorf("unexpected operation byte: %x", op)
		}
		if err != nil {
			return nil, err
		}
	}
}

func encodedLen(diffs []diffmatchpatch.Diff) int {
	var total int

	for _, d := range diffs {
		// Length of encoded data length
		total += int(math.Ceil(math.Log(float64(len(d.Text))) / math.Log(16)))

		// Op byte
		total++

		// Data
		if d.Type == diffmatchpatch.DiffInsert {
			total += len(d.Text)
		}
	}

	return total
}

func readOp(r *trackedReader) (int, byte, error) {
	s := make([]byte, 0, 10)

	for {
		c, err := r.ReadByte()
		if err != nil {
			return 0, 0, err
		}
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f':
			s = append(s, c)
			if len(s) > 9 {
				return 0, 0, fmt.Errorf("expected operation code, pos: %d", r.pos())
			}
		case c == OpCopy, c == OpInsert, c == OpDelete, c == OpCRC:
			if len(s) == 0 {
				return 0, 0, fmt.Errorf("missing operation length, pos: %d", r.pos())
			}
			l, err := strconv.ParseInt(string(s), 16, 64)
			if err != nil {
				return 0, 0, fmt.Errorf("error decoding length: %w, pos: %d", err, r.pos())
			}
			return int(l), c, nil

		default:
			return 0, 0, fmt.Errorf("error decoding operation %q, pos: %d", string(c), r.pos())
		}
	}
}

type trackedReader struct {
	*bytes.Reader
	bytesRead int
}

func newTrackedReader(b []byte) *trackedReader {
	return &trackedReader{
		Reader: bytes.NewReader(b),
	}
}

func (t *trackedReader) pos() int64 {
	return t.Size() - int64(t.Len())
}
