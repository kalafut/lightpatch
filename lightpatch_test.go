package lightpatch

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_lightpatch(t *testing.T) {
	t.Run("basic diff", func(t *testing.T) {
		a := []byte("The quick brown fox jumped over the lazy dog.")
		b := []byte("The quick brown cat jumped over the dog!")

		ar := bytes.NewReader(a)
		br := bytes.NewReader(b)

		var patchr bytes.Buffer
		err := MakePatch(ar, br, &patchr)
		assert.NoError(t, err)

		ar = bytes.NewReader(a)

		var c bytes.Buffer
		err = ApplyPatch(ar, &patchr, &c)
		assert.NoError(t, err)
		assert.Equal(t, b, c.Bytes())
	})

	t.Run("naive diff", func(t *testing.T) {
		a := make([]byte, 100)
		b := make([]byte, 100)
		rand.Read(a)
		rand.Read(b)

		ar := bytes.NewReader(a)
		br := bytes.NewReader(b)

		var patchr bytes.Buffer
		err := MakePatch(ar, br, &patchr)
		assert.NoError(t, err)

		// Check that we fell back to a naive diff (copying data) for this case of
		// "undiffable" random inputs. Without falling back to a naive diff, the
		// output is more than 150 bytes.
		assert.Equal(t, 107, patchr.Len())
	})

	t.Run("crc check", func(t *testing.T) {
		a := []byte("The quick brown fox jumped over the lazy dog.")
		b := []byte("The quick brown cat jumped over the dog!")

		ar := bytes.NewReader(a)
		br := bytes.NewReader(b)

		var patchr bytes.Buffer
		err := MakePatch(ar, br, &patchr)
		assert.NoError(t, err)

		// alter a to change the crc
		a[0] = 't'
		ar = bytes.NewReader(a)

		var c bytes.Buffer
		err = ApplyPatch(ar, &patchr, &c)
		assert.EqualError(t, err, ErrCRC.Error())
	})

	t.Run("extra patch bytes", func(t *testing.T) {
		a := []byte("The quick brown fox jumped over the lazy dog.")
		b := []byte("The quick brown cat jumped over the dog!")

		ar := bytes.NewReader(a)
		br := bytes.NewReader(b)

		var patchr bytes.Buffer
		err := MakePatch(ar, br, &patchr)
		assert.NoError(t, err)

		ar = bytes.NewReader(a)

		// add a byte to the patch, which isn't allowed after the CRC
		patchr.WriteByte(42)

		err = ApplyPatch(ar, &patchr, new(bytes.Buffer))
		assert.EqualError(t, err, ErrExtraData.Error())
	})
}
