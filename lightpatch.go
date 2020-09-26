// Package lightpatch generates and applies patch files. A description of the patch file
// format is included in the README.
package lightpatch

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"time"
)

const (
	OpCopy   byte = 'C'
	OpInsert byte = 'I'
	OpDelete byte = 'D'
	OpCRC    byte = 'K'

	DefaultTimeout = 5 * time.Second
)

var (
	ErrCRC       = errors.New("CRC mismatch")
	ErrExtraData = errors.New("unexpected data following CRC")
)

func MakePatch(before, after io.Reader, output io.Writer) error {
	return MakePatchTimeout(before, after, output, DefaultTimeout)
}

func MakePatchTimeout(before, after io.Reader, patch io.Writer, timeout time.Duration) error {
	beforeBytes, err := ioutil.ReadAll(before)
	if err != nil {
		return err
	}
	afterBytes, err := ioutil.ReadAll(after)
	if err != nil {
		return err
	}

	diffs := diffMain(beforeBytes, afterBytes, timeout)

	// If inputs are very different, the total size of the encoded diffs can be greater than just
	// outputting after bytes. We'll check whether this "naive" diff is actually shorter.
	naiveDiff := []diff{
		{
			Type: OpInsert,
			Text: afterBytes,
		},
	}

	if encodedLen(naiveDiff) < encodedLen(diffs) {
		diffs = naiveDiff
	}

	varintBuf := make([]byte, binary.MaxVarintLen64)

	for _, diff := range diffs {
		if _, err := patch.Write([]byte{diff.Type}); err != nil {
			return err
		}

		n := binary.PutUvarint(varintBuf, uint64(len(diff.Text)))
		if _, err := patch.Write(varintBuf[:n]); err != nil {
			return err
		}

		if diff.Type == OpInsert {
			if _, err := patch.Write(diff.Text); err != nil {
				return err
			}
		}
	}

	n := crc32.NewIEEE()
	n.Write(afterBytes)

	if _, err := patch.Write(n.Sum([]byte{OpCRC})); err != nil {
		return err
	}

	return nil
}

func ApplyPatch(before, patch io.Reader, after io.Writer) error {
	var crcRead bool
	var n = crc32.NewIEEE()

	after = io.MultiWriter(after, n)
	beforeBR := bufio.NewReader(before)
	patchBR := bufio.NewReader(patch)

	for {
		op, err := patchBR.ReadByte()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if crcRead {
			return ErrExtraData
		}

		var tl uint64
		if op != OpCRC {
			tl, err = binary.ReadUvarint(patchBR)
			if err != nil {
				return err
			}
		}

		switch op {
		case OpCopy:
			_, err := io.CopyN(after, beforeBR, int64(tl))
			if err != nil {
				return err
			}
		case OpInsert:
			_, err := io.CopyN(after, patchBR, int64(tl))
			if err != nil {
				return err
			}
		case OpDelete:
			_, err := beforeBR.Discard(int(tl))
			if err != nil {
				return err
			}
		case OpCRC:
			patchCRC := make([]byte, 4)
			_, err := io.ReadFull(patchBR, patchCRC)
			if err != nil {
				return err
			}

			if !bytes.Equal(patchCRC, n.Sum(nil)) {
				return ErrCRC
			}
			crcRead = true

		default:
			return fmt.Errorf("unexpected operation byte: %x", op)
		}
	}

	return nil
}

func encodedLen(diffs []diff) int {
	var total int

	for _, d := range diffs {
		// Op bytes
		total++

		// Size bytes. Copied from varint code
		x := len(d.Text)
		for x >= 0x80 {
			x >>= 7
			total++
		}
		total++

		// Data
		if d.Type == OpInsert {
			total += len(d.Text)
		}
	}

	return total
}
