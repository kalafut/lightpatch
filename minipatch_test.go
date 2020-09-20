// Copyright (c) 2012-2016 The go-diff authors. All rights reserved.
// https://github.com/sergi/go-diff
// See the included LICENSE file for license details.
//
// go-diff is a Go implementation of Google's diff, Match, and Patch library
// Original library is Copyright (c) 2006 Google Inc.
// http://code.google.com/p/google-diff-match-patch/

package minipatch

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// diffTest is the old diff structure in the original library.
// It is provided to retain compatibility with existing tests,
// which use string instead of []byte.
type diffTest struct {
	Type byte
	Text string
}

func (d diffTest) asDiff() diff {
	return diff{
		d.Type,
		[]byte(d.Text),
	}
}

func asDiffs(diffOlds []diffTest) []diff {
	diffs := []diff{}

	for _, d := range diffOlds {
		diffs = append(diffs, d.asDiff())
	}

	return diffs
}

func TestCommonPrefixLength(t *testing.T) {
	type TestCase struct {
		Text1 string
		Text2 string

		Expected int
	}

	for i, tc := range []TestCase{
		{"abc", "xyz", 0},
		{"1234abcdef", "1234xyz", 4},
		{"1234", "1234xyz", 4},
	} {
		actual := commonPrefixLength([]byte(tc.Text1), []byte(tc.Text2))
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}
}

func TestCommonSuffixLength(t *testing.T) {
	type TestCase struct {
		Text1 string
		Text2 string

		Expected int
	}

	for i, tc := range []TestCase{
		{"abc", "xyz", 0},
		{"abcdef1234", "xyz1234", 4},
		{"1234", "xyz1234", 4},
		{"123", "a3", 1},
	} {
		actual := commonSuffixLength([]byte(tc.Text1), []byte(tc.Text2))
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}
}

func TestDiffCommonOverlap(t *testing.T) {
	type TestCase struct {
		Name string

		Text1 string
		Text2 string

		Expected int
	}

	for i, tc := range []TestCase{
		{"Null", "", "abcd", 0},
		{"Whole", "abc", "abcd", 3},
		{"Null", "123456", "abcd", 0},
		{"Null", "123456xxx", "xxxabcd", 3},
		// Some overly clever languages (C#) may treat ligatures as equal to their component letters, e.g. U+FB01 == 'fi'
		{"Unicode", "fi", "\ufb01i", 0},
	} {
		actual := diffCommonOverlap([]byte(tc.Text1), []byte(tc.Text2))
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}
}

func TestDiffCleanupMerge(t *testing.T) {
	type TestCase struct {
		Name string

		Diffs []diffTest

		Expected []diffTest
	}

	for i, tc := range []TestCase{
		{
			"Null case",
			[]diffTest{},
			[]diffTest{},
		},
		{
			"No diff case",
			[]diffTest{{OpCopy, "a"}, {OpDelete, "b"}, {OpInsert, "c"}},
			[]diffTest{{OpCopy, "a"}, {OpDelete, "b"}, {OpInsert, "c"}},
		},
		{
			"Merge equalities",
			[]diffTest{{OpCopy, "a"}, {OpCopy, "b"}, {OpCopy, "c"}},
			[]diffTest{{OpCopy, "abc"}},
		},
		{
			"Merge deletions",
			[]diffTest{{OpDelete, "a"}, {OpDelete, "b"}, {OpDelete, "c"}},
			[]diffTest{{OpDelete, "abc"}},
		},
		{
			"Merge insertions",
			[]diffTest{{OpInsert, "a"}, {OpInsert, "b"}, {OpInsert, "c"}},
			[]diffTest{{OpInsert, "abc"}},
		},
		{
			"Merge interweave",
			[]diffTest{{OpDelete, "a"}, {OpInsert, "b"}, {OpDelete, "c"}, {OpInsert, "d"}, {OpCopy, "e"}, {OpCopy, "f"}},
			[]diffTest{{OpDelete, "ac"}, {OpInsert, "bd"}, {OpCopy, "ef"}},
		},
		{
			"Prefix and suffix detection",
			[]diffTest{{OpDelete, "a"}, {OpInsert, "abc"}, {OpDelete, "dc"}},
			[]diffTest{{OpCopy, "a"}, {OpDelete, "d"}, {OpInsert, "b"}, {OpCopy, "c"}},
		},
		{
			"Prefix and suffix detection with equalities",
			[]diffTest{{OpCopy, "x"}, {OpDelete, "a"}, {OpInsert, "abc"}, {OpDelete, "dc"}, {OpCopy, "y"}},
			[]diffTest{{OpCopy, "xa"}, {OpDelete, "d"}, {OpInsert, "b"}, {OpCopy, "cy"}},
		},
		{
			"Same test as above but with unicode (\u0101 will appear in diffs with at least 257 unique lines)",
			[]diffTest{{OpCopy, "x"}, {OpDelete, "\u0101"}, {OpInsert, "\u0101bc"}, {OpDelete, "dc"}, {OpCopy, "y"}},
			[]diffTest{{OpCopy, "x\u0101"}, {OpDelete, "d"}, {OpInsert, "b"}, {OpCopy, "cy"}},
		},
		{
			"Slide edit left",
			[]diffTest{{OpCopy, "a"}, {OpInsert, "ba"}, {OpCopy, "c"}},
			[]diffTest{{OpInsert, "ab"}, {OpCopy, "ac"}},
		},
		{
			"Slide edit right",
			[]diffTest{{OpCopy, "c"}, {OpInsert, "ab"}, {OpCopy, "a"}},
			[]diffTest{{OpCopy, "ca"}, {OpInsert, "ba"}},
		},
		{
			"Slide edit left recursive",
			[]diffTest{{OpCopy, "a"}, {OpDelete, "b"}, {OpCopy, "c"}, {OpDelete, "ac"}, {OpCopy, "x"}},
			[]diffTest{{OpDelete, "abc"}, {OpCopy, "acx"}},
		},
		{
			"Slide edit right recursive",
			[]diffTest{{OpCopy, "x"}, {OpDelete, "ca"}, {OpCopy, "c"}, {OpDelete, "b"}, {OpCopy, "a"}},
			[]diffTest{{OpCopy, "xca"}, {OpDelete, "cba"}},
		},
	} {
		actual := diffCleanupMerge(asDiffs(tc.Diffs))
		assert.Equal(t, asDiffs(tc.Expected), actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}
}

func TestDiffCleanupSemantic(t *testing.T) {
	type TestCase struct {
		Name string

		Diffs []diffTest

		Expected []diffTest
	}

	for i, tc := range []TestCase{
		{
			"Null case",
			[]diffTest{},
			[]diffTest{},
		},
		{
			"No elimination #1",
			[]diffTest{
				{OpDelete, "ab"},
				{OpInsert, "cd"},
				{OpCopy, "12"},
				{OpDelete, "e"},
			},
			[]diffTest{
				{OpDelete, "ab"},
				{OpInsert, "cd"},
				{OpCopy, "12"},
				{OpDelete, "e"},
			},
		},
		{
			"No elimination #2",
			[]diffTest{
				{OpDelete, "abc"},
				{OpInsert, "ABC"},
				{OpCopy, "1234"},
				{OpDelete, "wxyz"},
			},
			[]diffTest{
				{OpDelete, "abc"},
				{OpInsert, "ABC"},
				{OpCopy, "1234"},
				{OpDelete, "wxyz"},
			},
		},
		{
			"No elimination #3",
			[]diffTest{
				{OpCopy, "2016-09-01T03:07:1"},
				{OpInsert, "5.15"},
				{OpCopy, "4"},
				{OpDelete, "."},
				{OpCopy, "80"},
				{OpInsert, "0"},
				{OpCopy, "78"},
				{OpDelete, "3074"},
				{OpCopy, "1Z"},
			},
			[]diffTest{
				{OpCopy, "2016-09-01T03:07:1"},
				{OpInsert, "5.15"},
				{OpCopy, "4"},
				{OpDelete, "."},
				{OpCopy, "80"},
				{OpInsert, "0"},
				{OpCopy, "78"},
				{OpDelete, "3074"},
				{OpCopy, "1Z"},
			},
		},
		{
			"Simple elimination",
			[]diffTest{
				{OpDelete, "a"},
				{OpCopy, "b"},
				{OpDelete, "c"},
			},
			[]diffTest{
				{OpDelete, "abc"},
				{OpInsert, "b"},
			},
		},
		{
			"Backpass elimination",
			[]diffTest{
				{OpDelete, "ab"},
				{OpCopy, "cd"},
				{OpDelete, "e"},
				{OpCopy, "f"},
				{OpInsert, "g"},
			},
			[]diffTest{
				{OpDelete, "abcdef"},
				{OpInsert, "cdfg"},
			},
		},
		{
			"Multiple eliminations",
			[]diffTest{
				{OpInsert, "1"},
				{OpCopy, "A"},
				{OpDelete, "B"},
				{OpInsert, "2"},
				{OpCopy, "_"},
				{OpInsert, "1"},
				{OpCopy, "A"},
				{OpDelete, "B"},
				{OpInsert, "2"},
			},
			[]diffTest{
				{OpDelete, "AB_AB"},
				{OpInsert, "1A2_1A2"},
			},
		},
		{
			"No overlap elimination",
			[]diffTest{
				{OpDelete, "abcxx"},
				{OpInsert, "xxdef"},
			},
			[]diffTest{
				{OpDelete, "abcxx"},
				{OpInsert, "xxdef"},
			},
		},
		{
			"Overlap elimination",
			[]diffTest{
				{OpDelete, "abcxxx"},
				{OpInsert, "xxxdef"},
			},
			[]diffTest{
				{OpDelete, "abc"},
				{OpCopy, "xxx"},
				{OpInsert, "def"},
			},
		},
		{
			"Reverse overlap elimination",
			[]diffTest{
				{OpDelete, "xxxabc"},
				{OpInsert, "defxxx"},
			},
			[]diffTest{
				{OpInsert, "def"},
				{OpCopy, "xxx"},
				{OpDelete, "abc"},
			},
		},
		{
			"Two overlap eliminations",
			[]diffTest{
				{OpDelete, "abcd1212"},
				{OpInsert, "1212efghi"},
				{OpCopy, "----"},
				{OpDelete, "A3"},
				{OpInsert, "3BC"},
			},
			[]diffTest{
				{OpDelete, "abcd"},
				{OpCopy, "1212"},
				{OpInsert, "efghi"},
				{OpCopy, "----"},
				{OpDelete, "A"},
				{OpCopy, "3"},
				{OpInsert, "BC"},
			},
		},
		{
			"Test case for adapting diffCleanupSemantic to be equal to the Python version #19",
			[]diffTest{
				{OpCopy, "James McCarthy "},
				{OpDelete, "close to "},
				{OpCopy, "sign"},
				{OpDelete, "ing"},
				{OpInsert, "s"},
				{OpCopy, " new "},
				{OpDelete, "E"},
				{OpInsert, "fi"},
				{OpCopy, "ve"},
				{OpInsert, "-yea"},
				{OpCopy, "r"},
				{OpDelete, "ton"},
				{OpCopy, " deal"},
				{OpInsert, " at Everton"},
			},
			[]diffTest{
				{OpCopy, "James McCarthy "},
				{OpDelete, "close to "},
				{OpCopy, "sign"},
				{OpDelete, "ing"},
				{OpInsert, "s"},
				{OpCopy, " new "},
				{OpInsert, "five-year deal at "},
				{OpCopy, "Everton"},
				{OpDelete, " deal"},
			},
		},
	} {
		actual := diffCleanupSemantic(asDiffs(tc.Diffs))
		assert.Equal(t, asDiffs(tc.Expected), actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}
}

//func BenchmarkDiffCleanupSemantic(b *testing.B) {
//	s1, s2 := speedtestTexts()
//
//	dmp := New()
//
//	diffs := dmp.diffMain(s1, s2, false)
//
//	b.ResetTimer()
//
//	for i := 0; i < b.N; i++ {
//		dmp.diffCleanupSemantic(diffs)
//	}
//}

func TestDiffCleanupEfficiency(t *testing.T) {
	type TestCase struct {
		Name string

		Diffs []diffTest

		Expected []diffTest
	}

	for i, tc := range []TestCase{
		{
			"Null case",
			[]diffTest{},
			[]diffTest{},
		},
		{
			"No elimination",
			[]diffTest{
				{OpDelete, "ab"},
				{OpInsert, "12"},
				{OpCopy, "wxyz"},
				{OpDelete, "cd"},
				{OpInsert, "34"},
			},
			[]diffTest{
				{OpDelete, "ab"},
				{OpInsert, "12"},
				{OpCopy, "wxyz"},
				{OpDelete, "cd"},
				{OpInsert, "34"},
			},
		},
		{
			"Four-edit elimination",
			[]diffTest{
				{OpDelete, "ab"},
				{OpInsert, "12"},
				{OpCopy, "xyz"},
				{OpDelete, "cd"},
				{OpInsert, "34"},
			},
			[]diffTest{
				{OpDelete, "abxyzcd"},
				{OpInsert, "12xyz34"},
			},
		},
		{
			"Three-edit elimination",
			[]diffTest{
				{OpInsert, "12"},
				{OpCopy, "x"},
				{OpDelete, "cd"},
				{OpInsert, "34"},
			},
			[]diffTest{
				{OpDelete, "xcd"},
				{OpInsert, "12x34"},
			},
		},
		{
			"Backpass elimination",
			[]diffTest{
				{OpDelete, "ab"},
				{OpInsert, "12"},
				{OpCopy, "xy"},
				{OpInsert, "34"},
				{OpCopy, "z"},
				{OpDelete, "cd"},
				{OpInsert, "56"},
			},
			[]diffTest{
				{OpDelete, "abxyzcd"},
				{OpInsert, "12xy34z56"},
			},
		},
	} {
		actual := diffCleanupEfficiency(asDiffs(tc.Diffs))
		assert.Equal(t, asDiffs(tc.Expected), actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}
}

func TestDiffMain(t *testing.T) {
	type TestCase struct {
		Text1 string
		Text2 string

		Expected []diffTest
	}

	// Perform a trivial diff.
	for i, tc := range []TestCase{
		{
			"",
			"",
			nil,
		},
		{
			"abc",
			"abc",
			[]diffTest{{OpCopy, "abc"}},
		},
		{
			"abc",
			"ab123c",
			[]diffTest{{OpCopy, "ab"}, {OpInsert, "123"}, {OpCopy, "c"}},
		},
		{
			"a123bc",
			"abc",
			[]diffTest{{OpCopy, "a"}, {OpDelete, "123"}, {OpCopy, "bc"}},
		},
		{
			"abc",
			"a123b456c",
			[]diffTest{{OpCopy, "a"}, {OpInsert, "123"}, {OpCopy, "b"}, {OpInsert, "456"}, {OpCopy, "c"}},
		},
		{
			"a123b456c",
			"abc",
			[]diffTest{{OpCopy, "a"}, {OpDelete, "123"}, {OpCopy, "b"}, {OpDelete, "456"}, {OpCopy, "c"}},
		},
	} {
		actual := diffMain([]byte(tc.Text1), []byte(tc.Text2), 0)
		assert.Equal(t, asDiffs(tc.Expected), actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}

	for i, tc := range []TestCase{
		{
			"a",
			"b",
			[]diffTest{{OpDelete, "a"}, {OpInsert, "b"}},
		},
		{
			"Apples are a fruit.",
			"Bananas are also fruit.",
			[]diffTest{
				{OpDelete, "Apple"},
				{OpInsert, "Banana"},
				{OpCopy, "s are a"},
				{OpInsert, "lso"},
				{OpCopy, " fruit."},
			},
		},
		{
			"ax\t",
			"\u0680x\u0000",
			[]diffTest{
				{OpDelete, "a"},
				{OpInsert, "\u0680"},
				{OpCopy, "x"},
				{OpDelete, "\t"},
				{OpInsert, "\u0000"},
			},
		},
		{
			"1ayb2",
			"abxab",
			[]diffTest{
				{OpDelete, "1"},
				{OpCopy, "a"},
				{OpDelete, "y"},
				{OpCopy, "b"},
				{OpDelete, "2"},
				{OpInsert, "xab"},
			},
		},
		{
			"abcy",
			"xaxcxabc",
			[]diffTest{
				{OpInsert, "xaxcx"},
				{OpCopy, "abc"}, {OpDelete, "y"},
			},
		},
		{
			"ABCDa=bcd=efghijklmnopqrsEFGHIJKLMNOefg",
			"a-bcd-efghijklmnopqrs",
			[]diffTest{
				{OpDelete, "ABCD"},
				{OpCopy, "a"},
				{OpDelete, "="},
				{OpInsert, "-"},
				{OpCopy, "bcd"},
				{OpDelete, "="},
				{OpInsert, "-"},
				{OpCopy, "efghijklmnopqrs"},
				{OpDelete, "EFGHIJKLMNOefg"},
			},
		},
		{
			"a [[Pennsylvania]] and [[New",
			" and [[Pennsylvania]]",
			[]diffTest{
				{OpInsert, " "},
				{OpCopy, "a"},
				{OpInsert, "nd"},
				{OpCopy, " [[Pennsylvania]]"},
				{OpDelete, " and [[New"},
			},
		},
	} {
		actual := diffMain([]byte(tc.Text1), []byte(tc.Text2), 0)
		assert.Equal(t, asDiffs(tc.Expected), actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}

	// Test for invalid UTF-8 sequences
	//assert.Equal(t, []diff{
	//	{OpDelete, "��"},
	//}, dmp.diffMain("\xe0\xe5", "", false))
}

func TestDiffMainWithTimeout(t *testing.T) {
	timeout := 1000 * time.Millisecond

	a := "`Twas brillig, and the slithy toves\nDid gyre and gimble in the wabe:\nAll mimsy were the borogoves,\nAnd the mome raths outgrabe.\n"
	b := "I am the very model of a modern major general,\nI've information vegetable, animal, and mineral,\nI know the kings of England, and I quote the fights historical,\nFrom Marathon to Waterloo, in order categorical.\n"
	// Increase the text lengths by 1024 times to ensure a timeout.
	for x := 0; x < 13; x++ {
		a = a + a
		b = b + b
	}

	startTime := time.Now()
	diffMain([]byte(a), []byte(b), timeout)
	endTime := time.Now()

	delta := endTime.Sub(startTime)

	// Test that we took at least the timeout period.
	assert.True(t, delta >= timeout, fmt.Sprintf("%v !>= %v", delta, timeout))

	// Test that we didn't take forever (be very forgiving). Theoretically this test could fail very occasionally if the OS task swaps or locks up for a second at the wrong moment.
	assert.True(t, delta < (timeout*100), fmt.Sprintf("%v !< %v", delta, timeout*100))
}

func Test_minipatch(t *testing.T) {
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
		assert.Equal(t, 102, patchr.Len())
	})
}
