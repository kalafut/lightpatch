// Copyright (c) 2012-2016 The go-diff authors. All rights reserved.
// https://github.com/sergi/go-diff
// See the included LICENSE file for license details.
//
// go-diff is a Go implementation of Google's Diff, Match, and Patch library
// Original library is Copyright (c) 2006 Google Inc.
// http://code.google.com/p/google-diff-match-patch/

package minipatch

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// diffTest is the old Diff structure in the original library.
// It is provided to retain compatibility with existing tests,
// which use string instead of []byte.
type diffTest struct {
	Type Operation
	Text string
}

func (d diffTest) asDiff() Diff {
	return Diff{
		d.Type,
		[]byte(d.Text),
	}
}

func asDiffs(diffOlds []diffTest) []Diff {
	diffs := []Diff{}

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

	dmp := New()

	for i, tc := range []TestCase{
		{"Null", "", "abcd", 0},
		{"Whole", "abc", "abcd", 3},
		{"Null", "123456", "abcd", 0},
		{"Null", "123456xxx", "xxxabcd", 3},
		// Some overly clever languages (C#) may treat ligatures as equal to their component letters, e.g. U+FB01 == 'fi'
		{"Unicode", "fi", "\ufb01i", 0},
	} {
		actual := dmp.DiffCommonOverlap([]byte(tc.Text1), []byte(tc.Text2))
		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}
}

func TestDiffCleanupMerge(t *testing.T) {
	type TestCase struct {
		Name string

		Diffs []diffTest

		Expected []diffTest
	}

	dmp := New()

	for i, tc := range []TestCase{
		{
			"Null case",
			[]diffTest{},
			[]diffTest{},
		},
		{
			"No Diff case",
			[]diffTest{{DiffEqual, "a"}, {DiffDelete, "b"}, {DiffInsert, "c"}},
			[]diffTest{{DiffEqual, "a"}, {DiffDelete, "b"}, {DiffInsert, "c"}},
		},
		{
			"Merge equalities",
			[]diffTest{{DiffEqual, "a"}, {DiffEqual, "b"}, {DiffEqual, "c"}},
			[]diffTest{{DiffEqual, "abc"}},
		},
		{
			"Merge deletions",
			[]diffTest{{DiffDelete, "a"}, {DiffDelete, "b"}, {DiffDelete, "c"}},
			[]diffTest{{DiffDelete, "abc"}},
		},
		{
			"Merge insertions",
			[]diffTest{{DiffInsert, "a"}, {DiffInsert, "b"}, {DiffInsert, "c"}},
			[]diffTest{{DiffInsert, "abc"}},
		},
		{
			"Merge interweave",
			[]diffTest{{DiffDelete, "a"}, {DiffInsert, "b"}, {DiffDelete, "c"}, {DiffInsert, "d"}, {DiffEqual, "e"}, {DiffEqual, "f"}},
			[]diffTest{{DiffDelete, "ac"}, {DiffInsert, "bd"}, {DiffEqual, "ef"}},
		},
		{
			"Prefix and suffix detection",
			[]diffTest{{DiffDelete, "a"}, {DiffInsert, "abc"}, {DiffDelete, "dc"}},
			[]diffTest{{DiffEqual, "a"}, {DiffDelete, "d"}, {DiffInsert, "b"}, {DiffEqual, "c"}},
		},
		{
			"Prefix and suffix detection with equalities",
			[]diffTest{{DiffEqual, "x"}, {DiffDelete, "a"}, {DiffInsert, "abc"}, {DiffDelete, "dc"}, {DiffEqual, "y"}},
			[]diffTest{{DiffEqual, "xa"}, {DiffDelete, "d"}, {DiffInsert, "b"}, {DiffEqual, "cy"}},
		},
		{
			"Same test as above but with unicode (\u0101 will appear in diffs with at least 257 unique lines)",
			[]diffTest{{DiffEqual, "x"}, {DiffDelete, "\u0101"}, {DiffInsert, "\u0101bc"}, {DiffDelete, "dc"}, {DiffEqual, "y"}},
			[]diffTest{{DiffEqual, "x\u0101"}, {DiffDelete, "d"}, {DiffInsert, "b"}, {DiffEqual, "cy"}},
		},
		{
			"Slide edit left",
			[]diffTest{{DiffEqual, "a"}, {DiffInsert, "ba"}, {DiffEqual, "c"}},
			[]diffTest{{DiffInsert, "ab"}, {DiffEqual, "ac"}},
		},
		{
			"Slide edit right",
			[]diffTest{{DiffEqual, "c"}, {DiffInsert, "ab"}, {DiffEqual, "a"}},
			[]diffTest{{DiffEqual, "ca"}, {DiffInsert, "ba"}},
		},
		{
			"Slide edit left recursive",
			[]diffTest{{DiffEqual, "a"}, {DiffDelete, "b"}, {DiffEqual, "c"}, {DiffDelete, "ac"}, {DiffEqual, "x"}},
			[]diffTest{{DiffDelete, "abc"}, {DiffEqual, "acx"}},
		},
		{
			"Slide edit right recursive",
			[]diffTest{{DiffEqual, "x"}, {DiffDelete, "ca"}, {DiffEqual, "c"}, {DiffDelete, "b"}, {DiffEqual, "a"}},
			[]diffTest{{DiffEqual, "xca"}, {DiffDelete, "cba"}},
		},
	} {
		actual := dmp.diffCleanupMerge(asDiffs(tc.Diffs))
		assert.Equal(t, asDiffs(tc.Expected), actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}
}

func TestDiffCleanupSemantic(t *testing.T) {
	type TestCase struct {
		Name string

		Diffs []diffTest

		Expected []diffTest
	}

	dmp := New()

	for i, tc := range []TestCase{
		{
			"Null case",
			[]diffTest{},
			[]diffTest{},
		},
		{
			"No elimination #1",
			[]diffTest{
				{DiffDelete, "ab"},
				{DiffInsert, "cd"},
				{DiffEqual, "12"},
				{DiffDelete, "e"},
			},
			[]diffTest{
				{DiffDelete, "ab"},
				{DiffInsert, "cd"},
				{DiffEqual, "12"},
				{DiffDelete, "e"},
			},
		},
		{
			"No elimination #2",
			[]diffTest{
				{DiffDelete, "abc"},
				{DiffInsert, "ABC"},
				{DiffEqual, "1234"},
				{DiffDelete, "wxyz"},
			},
			[]diffTest{
				{DiffDelete, "abc"},
				{DiffInsert, "ABC"},
				{DiffEqual, "1234"},
				{DiffDelete, "wxyz"},
			},
		},
		{
			"No elimination #3",
			[]diffTest{
				{DiffEqual, "2016-09-01T03:07:1"},
				{DiffInsert, "5.15"},
				{DiffEqual, "4"},
				{DiffDelete, "."},
				{DiffEqual, "80"},
				{DiffInsert, "0"},
				{DiffEqual, "78"},
				{DiffDelete, "3074"},
				{DiffEqual, "1Z"},
			},
			[]diffTest{
				{DiffEqual, "2016-09-01T03:07:1"},
				{DiffInsert, "5.15"},
				{DiffEqual, "4"},
				{DiffDelete, "."},
				{DiffEqual, "80"},
				{DiffInsert, "0"},
				{DiffEqual, "78"},
				{DiffDelete, "3074"},
				{DiffEqual, "1Z"},
			},
		},
		{
			"Simple elimination",
			[]diffTest{
				{DiffDelete, "a"},
				{DiffEqual, "b"},
				{DiffDelete, "c"},
			},
			[]diffTest{
				{DiffDelete, "abc"},
				{DiffInsert, "b"},
			},
		},
		{
			"Backpass elimination",
			[]diffTest{
				{DiffDelete, "ab"},
				{DiffEqual, "cd"},
				{DiffDelete, "e"},
				{DiffEqual, "f"},
				{DiffInsert, "g"},
			},
			[]diffTest{
				{DiffDelete, "abcdef"},
				{DiffInsert, "cdfg"},
			},
		},
		{
			"Multiple eliminations",
			[]diffTest{
				{DiffInsert, "1"},
				{DiffEqual, "A"},
				{DiffDelete, "B"},
				{DiffInsert, "2"},
				{DiffEqual, "_"},
				{DiffInsert, "1"},
				{DiffEqual, "A"},
				{DiffDelete, "B"},
				{DiffInsert, "2"},
			},
			[]diffTest{
				{DiffDelete, "AB_AB"},
				{DiffInsert, "1A2_1A2"},
			},
		},
		{
			"No overlap elimination",
			[]diffTest{
				{DiffDelete, "abcxx"},
				{DiffInsert, "xxdef"},
			},
			[]diffTest{
				{DiffDelete, "abcxx"},
				{DiffInsert, "xxdef"},
			},
		},
		{
			"Overlap elimination",
			[]diffTest{
				{DiffDelete, "abcxxx"},
				{DiffInsert, "xxxdef"},
			},
			[]diffTest{
				{DiffDelete, "abc"},
				{DiffEqual, "xxx"},
				{DiffInsert, "def"},
			},
		},
		{
			"Reverse overlap elimination",
			[]diffTest{
				{DiffDelete, "xxxabc"},
				{DiffInsert, "defxxx"},
			},
			[]diffTest{
				{DiffInsert, "def"},
				{DiffEqual, "xxx"},
				{DiffDelete, "abc"},
			},
		},
		{
			"Two overlap eliminations",
			[]diffTest{
				{DiffDelete, "abcd1212"},
				{DiffInsert, "1212efghi"},
				{DiffEqual, "----"},
				{DiffDelete, "A3"},
				{DiffInsert, "3BC"},
			},
			[]diffTest{
				{DiffDelete, "abcd"},
				{DiffEqual, "1212"},
				{DiffInsert, "efghi"},
				{DiffEqual, "----"},
				{DiffDelete, "A"},
				{DiffEqual, "3"},
				{DiffInsert, "BC"},
			},
		},
		{
			"Test case for adapting DiffCleanupSemantic to be equal to the Python version #19",
			[]diffTest{
				{DiffEqual, "James McCarthy "},
				{DiffDelete, "close to "},
				{DiffEqual, "sign"},
				{DiffDelete, "ing"},
				{DiffInsert, "s"},
				{DiffEqual, " new "},
				{DiffDelete, "E"},
				{DiffInsert, "fi"},
				{DiffEqual, "ve"},
				{DiffInsert, "-yea"},
				{DiffEqual, "r"},
				{DiffDelete, "ton"},
				{DiffEqual, " deal"},
				{DiffInsert, " at Everton"},
			},
			[]diffTest{
				{DiffEqual, "James McCarthy "},
				{DiffDelete, "close to "},
				{DiffEqual, "sign"},
				{DiffDelete, "ing"},
				{DiffInsert, "s"},
				{DiffEqual, " new "},
				{DiffInsert, "five-year deal at "},
				{DiffEqual, "Everton"},
				{DiffDelete, " deal"},
			},
		},
	} {
		actual := dmp.DiffCleanupSemantic(asDiffs(tc.Diffs))
		assert.Equal(t, asDiffs(tc.Expected), actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}
}

//func BenchmarkDiffCleanupSemantic(b *testing.B) {
//	s1, s2 := speedtestTexts()
//
//	dmp := New()
//
//	diffs := dmp.DiffMain(s1, s2, false)
//
//	b.ResetTimer()
//
//	for i := 0; i < b.N; i++ {
//		dmp.DiffCleanupSemantic(diffs)
//	}
//}

func TestDiffCleanupEfficiency(t *testing.T) {
	type TestCase struct {
		Name string

		Diffs []diffTest

		Expected []diffTest
	}

	dmp := New()
	dmp.DiffEditCost = 4

	for i, tc := range []TestCase{
		{
			"Null case",
			[]diffTest{},
			[]diffTest{},
		},
		{
			"No elimination",
			[]diffTest{
				{DiffDelete, "ab"},
				{DiffInsert, "12"},
				{DiffEqual, "wxyz"},
				{DiffDelete, "cd"},
				{DiffInsert, "34"},
			},
			[]diffTest{
				{DiffDelete, "ab"},
				{DiffInsert, "12"},
				{DiffEqual, "wxyz"},
				{DiffDelete, "cd"},
				{DiffInsert, "34"},
			},
		},
		{
			"Four-edit elimination",
			[]diffTest{
				{DiffDelete, "ab"},
				{DiffInsert, "12"},
				{DiffEqual, "xyz"},
				{DiffDelete, "cd"},
				{DiffInsert, "34"},
			},
			[]diffTest{
				{DiffDelete, "abxyzcd"},
				{DiffInsert, "12xyz34"},
			},
		},
		{
			"Three-edit elimination",
			[]diffTest{
				{DiffInsert, "12"},
				{DiffEqual, "x"},
				{DiffDelete, "cd"},
				{DiffInsert, "34"},
			},
			[]diffTest{
				{DiffDelete, "xcd"},
				{DiffInsert, "12x34"},
			},
		},
		{
			"Backpass elimination",
			[]diffTest{
				{DiffDelete, "ab"},
				{DiffInsert, "12"},
				{DiffEqual, "xy"},
				{DiffInsert, "34"},
				{DiffEqual, "z"},
				{DiffDelete, "cd"},
				{DiffInsert, "56"},
			},
			[]diffTest{
				{DiffDelete, "abxyzcd"},
				{DiffInsert, "12xy34z56"},
			},
		},
	} {
		actual := dmp.diffCleanupEfficiency(asDiffs(tc.Diffs))
		assert.Equal(t, asDiffs(tc.Expected), actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}

	dmp.DiffEditCost = 5

	for i, tc := range []TestCase{
		{
			"High cost elimination",
			[]diffTest{
				{DiffDelete, "ab"},
				{DiffInsert, "12"},
				{DiffEqual, "wxyz"},
				{DiffDelete, "cd"},
				{DiffInsert, "34"},
			},
			[]diffTest{
				{DiffDelete, "abwxyzcd"},
				{DiffInsert, "12wxyz34"},
			},
		},
	} {
		actual := dmp.diffCleanupEfficiency(asDiffs(tc.Diffs))
		assert.Equal(t, asDiffs(tc.Expected), actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
	}
}

//func TestDiffPrettyHtml(t *testing.T) {
//	type TestCase struct {
//		Diffs []Diff
//
//		Expected string
//	}
//
//	dmp := New()
//
//	for i, tc := range []TestCase{
//		{
//			Diffs: []Diff{
//				{DiffEqual, "a\n"},
//				{DiffDelete, "<B>b</B>"},
//				{DiffInsert, "c&d"},
//			},
//
//			Expected: "<span>a&para;<br></span><del style=\"background:#ffe6e6;\">&lt;B&gt;b&lt;/B&gt;</del><ins style=\"background:#e6ffe6;\">c&amp;d</ins>",
//		},
//	} {
//		actual := dmp.DiffPrettyHtml(tc.Diffs)
//		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
//	}
//}

//func TestDiffPrettyText(t *testing.T) {
//	type TestCase struct {
//		Diffs []Diff
//
//		Expected string
//	}
//
//	dmp := New()
//
//	for i, tc := range []TestCase{
//		{
//			Diffs: []Diff{
//				{DiffEqual, "a\n"},
//				{DiffDelete, "<B>b</B>"},
//				{DiffInsert, "c&d"},
//			},
//
//			Expected: "a\n\x1b[31m<B>b</B>\x1b[0m\x1b[32mc&d\x1b[0m",
//		},
//	} {
//		actual := dmp.DiffPrettyText(tc.Diffs)
//		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
//	}
//}

//func TestDiffDelta(t *testing.T) {
//	type TestCase struct {
//		Name string
//
//		Text  string
//		Delta string
//
//		ErrorMessagePrefix string
//	}
//
//	dmp := New()
//
//	for i, tc := range []TestCase{
//		{"Delta shorter than text", "jumps over the lazyx", "=4\t-1\t+ed\t=6\t-3\t+a\t=5\t+old dog", "Delta length (19) is different from source text length (20)"},
//		{"Delta longer than text", "umps over the lazy", "=4\t-1\t+ed\t=6\t-3\t+a\t=5\t+old dog", "Delta length (19) is different from source text length (18)"},
//		{"Invalid URL escaping", "", "+%c3%xy", "invalid URL escape \"%xy\""},
//		{"Invalid UTF-8 sequence", "", "+%c3xy", "invalid UTF-8 token: \"\\xc3xy\""},
//		{"Invalid diff operation", "", "a", "Invalid diff operation in DiffFromDelta: a"},
//		{"Invalid diff syntax", "", "-", "strconv.ParseInt: parsing \"\": invalid syntax"},
//		{"Negative number in delta", "", "--1", "Negative number in DiffFromDelta: -1"},
//		{"Empty case", "", "", ""},
//	} {
//		diffs, err := dmp.DiffFromDelta(tc.Text, tc.Delta)
//		msg := fmt.Sprintf("Test case #%d, %s", i, tc.Name)
//		if tc.ErrorMessagePrefix == "" {
//			assert.Nil(t, err, msg)
//			assert.Nil(t, diffs, msg)
//		} else {
//			e := err.Error()
//			if strings.HasPrefix(e, tc.ErrorMessagePrefix) {
//				e = tc.ErrorMessagePrefix
//			}
//			assert.Nil(t, diffs, msg)
//			assert.Equal(t, tc.ErrorMessagePrefix, e, msg)
//		}
//	}
//
//	// Convert a diff into delta string.
//	diffsOld := []diffTest{
//		{DiffEqual, "jump"},
//		{DiffDelete, "s"},
//		{DiffInsert, "ed"},
//		{DiffEqual, " over "},
//		{DiffDelete, "the"},
//		{DiffInsert, "a"},
//		{DiffEqual, " lazy"},
//		{DiffInsert, "old dog"},
//	}
//
//	var diffs []Diff
//	for _, d := range diffsOld {
//		diffs = append(diffs, d.asDiff())
//	}
//
//	text1 := dmp.DiffText1(diffs)
//	assert.Equal(t, "jumps over the lazy", text1)
//
//	delta := dmp.DiffToDelta(diffs)
//	assert.Equal(t, "=4\t-1\t+ed\t=6\t-3\t+a\t=5\t+old dog", delta)
//
//	// Convert delta string into a diff.
//	deltaDiffs, err := dmp.DiffFromDelta(text1, delta)
//	assert.Equal(t, diffs, deltaDiffs)
//
//	// Test deltas with special characters.
//	diffs = []Diff{
//		{DiffEqual, "\u0680 \x00 \t %"},
//		{DiffDelete, "\u0681 \x01 \n ^"},
//		{DiffInsert, "\u0682 \x02 \\ |"},
//	}
//	text1 = dmp.DiffText1(diffs)
//	assert.Equal(t, "\u0680 \x00 \t %\u0681 \x01 \n ^", text1)
//
//	// Lowercase, due to UrlEncode uses lower.
//	delta = dmp.DiffToDelta(diffs)
//	assert.Equal(t, "=7\t-7\t+%DA%82 %02 %5C %7C", delta)
//
//	deltaDiffs, err = dmp.DiffFromDelta(text1, delta)
//	assert.Equal(t, diffs, deltaDiffs)
//	assert.Nil(t, err)
//
//	// Verify pool of unchanged characters.
//	diffs = []Diff{
//		{DiffInsert, "A-Z a-z 0-9 - _ . ! ~ * ' ( ) ; / ? : @ & = + $ , # "},
//	}
//
//	delta = dmp.DiffToDelta(diffs)
//	assert.Equal(t, "+A-Z a-z 0-9 - _ . ! ~ * ' ( ) ; / ? : @ & = + $ , # ", delta, "Unchanged characters.")
//
//	// Convert delta string into a diff.
//	deltaDiffs, err = dmp.DiffFromDelta("", delta)
//	assert.Equal(t, diffs, deltaDiffs)
//	assert.Nil(t, err)
//}

//func TestDiffXIndex(t *testing.T) {
//	type TestCase struct {
//		Name string
//
//		Diffs    []Diff
//		Location int
//
//		Expected int
//	}
//
//	dmp := New()
//
//	for i, tc := range []TestCase{
//		{"Translation on equality", []Diff{{DiffDelete, "a"}, {DiffInsert, "1234"}, {DiffEqual, "xyz"}}, 2, 5},
//		{"Translation on deletion", []Diff{{DiffEqual, "a"}, {DiffDelete, "1234"}, {DiffEqual, "xyz"}}, 3, 1},
//	} {
//		actual := dmp.DiffXIndex(tc.Diffs, tc.Location)
//		assert.Equal(t, tc.Expected, actual, fmt.Sprintf("Test case #%d, %s", i, tc.Name))
//	}
//}

func TestDiffMain(t *testing.T) {
	type TestCase struct {
		Text1 string
		Text2 string

		Expected []diffTest
	}

	dmp := New()

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
			[]diffTest{{DiffEqual, "abc"}},
		},
		{
			"abc",
			"ab123c",
			[]diffTest{{DiffEqual, "ab"}, {DiffInsert, "123"}, {DiffEqual, "c"}},
		},
		{
			"a123bc",
			"abc",
			[]diffTest{{DiffEqual, "a"}, {DiffDelete, "123"}, {DiffEqual, "bc"}},
		},
		{
			"abc",
			"a123b456c",
			[]diffTest{{DiffEqual, "a"}, {DiffInsert, "123"}, {DiffEqual, "b"}, {DiffInsert, "456"}, {DiffEqual, "c"}},
		},
		{
			"a123b456c",
			"abc",
			[]diffTest{{DiffEqual, "a"}, {DiffDelete, "123"}, {DiffEqual, "b"}, {DiffDelete, "456"}, {DiffEqual, "c"}},
		},
	} {
		actual := dmp.DiffMain([]byte(tc.Text1), []byte(tc.Text2), false)
		assert.Equal(t, asDiffs(tc.Expected), actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}

	// Perform a real diff and switch off the timeout.
	dmp.DiffTimeout = 0

	for i, tc := range []TestCase{
		{
			"a",
			"b",
			[]diffTest{{DiffDelete, "a"}, {DiffInsert, "b"}},
		},
		{
			"Apples are a fruit.",
			"Bananas are also fruit.",
			[]diffTest{
				{DiffDelete, "Apple"},
				{DiffInsert, "Banana"},
				{DiffEqual, "s are a"},
				{DiffInsert, "lso"},
				{DiffEqual, " fruit."},
			},
		},
		{
			"ax\t",
			"\u0680x\u0000",
			[]diffTest{
				{DiffDelete, "a"},
				{DiffInsert, "\u0680"},
				{DiffEqual, "x"},
				{DiffDelete, "\t"},
				{DiffInsert, "\u0000"},
			},
		},
		{
			"1ayb2",
			"abxab",
			[]diffTest{
				{DiffDelete, "1"},
				{DiffEqual, "a"},
				{DiffDelete, "y"},
				{DiffEqual, "b"},
				{DiffDelete, "2"},
				{DiffInsert, "xab"},
			},
		},
		{
			"abcy",
			"xaxcxabc",
			[]diffTest{
				{DiffInsert, "xaxcx"},
				{DiffEqual, "abc"}, {DiffDelete, "y"},
			},
		},
		{
			"ABCDa=bcd=efghijklmnopqrsEFGHIJKLMNOefg",
			"a-bcd-efghijklmnopqrs",
			[]diffTest{
				{DiffDelete, "ABCD"},
				{DiffEqual, "a"},
				{DiffDelete, "="},
				{DiffInsert, "-"},
				{DiffEqual, "bcd"},
				{DiffDelete, "="},
				{DiffInsert, "-"},
				{DiffEqual, "efghijklmnopqrs"},
				{DiffDelete, "EFGHIJKLMNOefg"},
			},
		},
		{
			"a [[Pennsylvania]] and [[New",
			" and [[Pennsylvania]]",
			[]diffTest{
				{DiffInsert, " "},
				{DiffEqual, "a"},
				{DiffInsert, "nd"},
				{DiffEqual, " [[Pennsylvania]]"},
				{DiffDelete, " and [[New"},
			},
		},
	} {
		actual := dmp.DiffMain([]byte(tc.Text1), []byte(tc.Text2), false)
		assert.Equal(t, asDiffs(tc.Expected), actual, fmt.Sprintf("Test case #%d, %#v", i, tc))
	}

	// Test for invalid UTF-8 sequences
	//assert.Equal(t, []Diff{
	//	{DiffDelete, "��"},
	//}, dmp.DiffMain("\xe0\xe5", "", false))
}

func TestDiffMainWithTimeout(t *testing.T) {
	dmp := New()
	dmp.DiffTimeout = 200 * time.Millisecond

	a := "`Twas brillig, and the slithy toves\nDid gyre and gimble in the wabe:\nAll mimsy were the borogoves,\nAnd the mome raths outgrabe.\n"
	b := "I am the very model of a modern major general,\nI've information vegetable, animal, and mineral,\nI know the kings of England, and I quote the fights historical,\nFrom Marathon to Waterloo, in order categorical.\n"
	// Increase the text lengths by 1024 times to ensure a timeout.
	for x := 0; x < 13; x++ {
		a = a + a
		b = b + b
	}

	startTime := time.Now()
	dmp.DiffMain([]byte(a), []byte(b), true)
	endTime := time.Now()

	delta := endTime.Sub(startTime)

	// Test that we took at least the timeout period.
	assert.True(t, delta >= dmp.DiffTimeout, fmt.Sprintf("%v !>= %v", delta, dmp.DiffTimeout))

	// Test that we didn't take forever (be very forgiving). Theoretically this test could fail very occasionally if the OS task swaps or locks up for a second at the wrong moment.
	assert.True(t, delta < (dmp.DiffTimeout*100), fmt.Sprintf("%v !< %v", delta, dmp.DiffTimeout*100))
}

func Test_jpatch(t *testing.T) {
	a := []byte("The quick brown fox jumped over the lazy dog.")
	b := []byte("The quick brown cat jumped over the dog!")

	patch := MakePatch(a, b)

	c, err := ApplyPatch(a, patch)
	assert.NoError(t, err)
	assert.Equal(t, b, c)
}
