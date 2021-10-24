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

	t.Run("corrupt diff", func(t *testing.T) {
		a := []byte("The quick brown fox jumped over the lazy dog.")
		b := []byte("The quick brown cat jumped over the dog!")

		patch := MakePatch(a, b)

		a[10]++

		_, err := ApplyPatch(a, patch)
		assert.EqualError(t, err, ErrCRC.Error())
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
		assert.Equal(t, 100+3+9, len(patch))
	})

	t.Run("format test", func(t *testing.T) {
		a := "The quick brown fox jumped over the lazy dog"
		b := "The quick brown fox leaped over the lazy dog🎉"

		patch := MakePatch([]byte(a), []byte(b))

		exp := "14C3D3Ilea15C4I🎉40763bb0K"
		assert.Equal(t, exp, string(patch))
	})
}
