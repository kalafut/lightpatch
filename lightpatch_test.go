package lightpatch

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_lightpatch(t *testing.T) {
	t.Run("basic diff", func(t *testing.T) {
		a := []byte("The quick brown fox jumped over the lazy dog.")
		b := []byte("The quick brown cat jumped over the dog!")

		patch := MakePatch(a, b)

		after, err := ApplyPatch(a, patch)
		assert.NoError(t, err)
		assert.Equal(t, b, after)
	})

	t.Run("CRC options", func(t *testing.T) {
		a := []byte("The quick brown fox jumped over the lazy dog.")
		b := []byte("The quick brown cat jumped over the dog!")

		patch := MakePatch(a, b)

		// base case of default options and no corruption
		_, err := ApplyPatch(a, patch)
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
		patch = MakePatch(a, b, WithNoCRC())
		a[10]++

		_, err = ApplyPatch(a, patch)
		assert.NoError(t, err)
	})

	t.Run("naive diff", func(t *testing.T) {
		a := make([]byte, 100)
		b := make([]byte, 100)
		rand.Read(a)
		rand.Read(b)

		patch := MakePatch(a, b)

		// Check that we fell back to a naive diff (copying data) for this case of
		// "undiffable" random inputs. The length will be 2 hex digits and an op code,
		// plus 8 CRC bytes and an op code, plus the original length.
		assert.Equal(t, 1+100+3+9, len(patch))
	})

	t.Run("format test", func(t *testing.T) {
		a := "The quick brown fox jumped over the lazy dog"
		b := "The quick brown fox leaped over the lazy dog🎉"

		patch := MakePatch([]byte(a), []byte(b))

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
}
