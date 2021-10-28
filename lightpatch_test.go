package lightpatch

import (
	"encoding/hex"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_lightpatch(t *testing.T) {
	t.Run("basic diff", func(t *testing.T) {
		a := []byte("The quick brown fox jumped over the lazy dog.")
		b := []byte("The quick brown cat jumped over the dog!")

		patch, err := MakePatch(a, b)
		assert.NoError(t, err)

		after, err := ApplyPatch(a, patch)
		assert.NoError(t, err)
		assert.Equal(t, b, after)
	})

	t.Run("CRC options", func(t *testing.T) {
		a := []byte("The quick brown fox jumped over the lazy dog.")
		b := []byte("The quick brown cat jumped over the dog!")

		patch, err := MakePatch(a, b)
		assert.NoError(t, err)

		// base case of default options and no corruption
		_, err = ApplyPatch(a, patch)
		assert.NoError(t, err)

		// corrupt input
		a[10]++

		// default case will error on mismatched CRC
		_, err = ApplyPatch(a, patch)
		assert.EqualError(t, err, ErrCRC.Error())

		// ignore CRC checking
		_, err = ApplyPatch(a, patch, WithNoCRC())
		assert.NoError(t, err)

		a = []byte("The quick brown fox jumped over the lazy dog.")
		b = []byte("The quick brown cat jumped over the dog!")

		// check that disabling CRC on generation works
		patch, err = MakePatch(a, b, WithNoCRC())
		assert.NoError(t, err)
		a[10]++

		_, err = ApplyPatch(a, patch)
		assert.NoError(t, err)
	})

	t.Run("naive diff", func(t *testing.T) {
		a := make([]byte, 100)
		b := make([]byte, 100)
		rand.Read(a)
		rand.Read(b)

		patch, err := MakePatch([]byte(hex.EncodeToString(a)), []byte(hex.EncodeToString(b)))
		assert.NoError(t, err)

		// Check that we fell back to a naive diff (copying data) for this case of
		// "undiffable" random inputs. The length will be 2 hex digits and an op code,
		// plus 8 CRC bytes and an op code, plus the original length.
		assert.Equal(t, 1+200+3+9, len(patch))
	})

	t.Run("format test", func(t *testing.T) {
		a := "The quick brown fox jumped over the lazy dog"
		b := "The quick brown fox leaped over the lazy dog🎉"

		patch, err := MakePatch([]byte(a), []byte(b))
		assert.NoError(t, err)

		exp := "A14C3D3Ilea15C4I🎉40763bb0K"
		assert.Equal(t, exp, string(patch))
	})

	t.Run("bad patch: excessive length", func(t *testing.T) {
		a := "The quick brown fox jumped over the lazy dog"

		patch := "2IZZ19239a8234C3D3Ilea15C4I🎉40763bb0K"

		_, err := ApplyPatch([]byte(a), []byte(patch))
		assert.Error(t, err)
	})

	t.Run("bad patch: missing version", func(t *testing.T) {
		a := "The quick brown fox jumped over the lazy dog"

		patch := "2IZZ19239a8234C3D3Ilea15C4I🎉40763bb0K"

		_, err := ApplyPatch([]byte(a), []byte(patch))
		assert.EqualError(t, err, "unknown version '2'")
	})

	t.Run("detect non-utf8 data", func(t *testing.T) {
		bad := []byte("\xff\xff\xff")
		good := []byte("hello")

		_, err := MakePatch(bad, good)
		assert.EqualError(t, err, "non-utf8 data in 'before' data")

		_, err = MakePatch(good, bad)
		assert.EqualError(t, err, "non-utf8 data in 'after' data")
	})

	t.Run("patch binary data", func(t *testing.T) {
		a := make([]byte, 100)
		b := make([]byte, 98)
		rand.Read(a)
		copy(b, a)
		b[67]++

		// patch of binary data should fail
		_, err := MakePatch(a, b)
		assert.EqualError(t, err, "non-utf8 data in 'before' data")

		// but base64 encoded should work
		patch, err := MakePatch(a, b, WithBinary()) //, WithNoCRC())
		assert.NoError(t, err)

		out, err := ApplyPatch(a, patch, WithBinary())
		assert.NoError(t, err)
		assert.Equal(t, b, out)
	})
}
