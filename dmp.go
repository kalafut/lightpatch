// The algorithms and tests in lightpatch were largely adapted from the go-diff
// library, which in turn was derived from the Diff-Map-Patch library. The
// original copyright is retained:
//
// Copyright (c) 2012-2016 The go-diff authors. All rights reserved.
// https://github.com/sergi/go-diff
// See the included LICENSE file for license details.
//
// go-diff is a Go implementation of Google's diff, Match, and Patch library
// Original library is Copyright (c) 2006 Google Inc.
// http://code.google.com/p/google-diff-match-patch/
package lightpatch

import (
	"bytes"
	"math"
	"time"
)

// diff represents one diff operation
type diff struct {
	Type byte
	Text []byte
}

const diffEditCost = 4

func diffMain(text1, text2 []byte, timeout time.Duration) []diff {
	var deadline time.Time
	if timeout > 0 {
		deadline = time.Now().Add(timeout)
	}

	return diffMainBytes(text1, text2, deadline)
}

func diffMainBytes(text1, text2 []byte, deadline time.Time) []diff {
	if bytes.Equal(text1, text2) {
		diffs := []diff{}
		if len(text1) > 0 {
			diffs = append(diffs, diff{OpCopy, clone(text1)})
		}
		return diffs
	}
	// Trim off common prefix (speedup).
	commonlength := commonPrefixLength(text1, text2)
	commonprefix := text1[:commonlength]
	text1 = text1[commonlength:]
	text2 = text2[commonlength:]

	// Trim off common suffix (speedup).
	commonlength = commonSuffixLength(text1, text2)
	commonsuffix := text1[len(text1)-commonlength:]
	text1 = text1[:len(text1)-commonlength]
	text2 = text2[:len(text2)-commonlength]

	// Compute the diff on the middle block.
	diffs := diffCompute(text1, text2, deadline)

	// Restore the prefix and suffix.
	if len(commonprefix) != 0 {
		diffs = append([]diff{diff{OpCopy, clone(commonprefix)}}, diffs...)
	}
	if len(commonsuffix) != 0 {
		diffs = append(diffs, diff{OpCopy, clone(commonsuffix)})
	}

	return diffCleanupMerge(diffs)
}

// diffCompute finds the differences between two rune slices.  Assumes that the texts do not have any common prefix or suffix.
func diffCompute(text1, text2 []byte, deadline time.Time) []diff {
	diffs := []diff{}
	if len(text1) == 0 {
		// Just add some text (speedup).
		return append(diffs, diff{OpInsert, clone(text2)})
	} else if len(text2) == 0 {
		// Just delete some text (speedup).
		return append(diffs, diff{OpDelete, clone(text1)})
	}

	var longtext, shorttext []byte
	if len(text1) > len(text2) {
		longtext = text1
		shorttext = text2
	} else {
		longtext = text2
		shorttext = text1
	}

	if i := bytes.Index(longtext, shorttext); i != -1 {
		op := OpInsert
		// Swap insertions for deletions if diff is reversed.
		if len(text1) > len(text2) {
			op = OpDelete
		}
		// Shorter text is inside the longer text (speedup).
		return []diff{
			diff{op, clone(longtext[:i])},
			diff{OpCopy, clone(shorttext)},
			diff{op, clone(longtext[i+len(shorttext):])},
		}
	} else if len(shorttext) == 1 {
		// Single character string.
		// After the previous speedup, the character can't be an equality.
		return []diff{
			diff{OpDelete, clone(text1)},
			diff{OpInsert, clone(text2)},
		}
		// Check to see if the problem can be split in two.
	} else if hm := diffHalfMatch(text1, text2, deadline.IsZero()); hm != nil {
		// A half-match was found, sort out the return data.
		text1A := hm[0]
		text1B := hm[1]
		text2A := hm[2]
		text2B := hm[3]
		midCommon := hm[4]
		// Send both pairs off for separate processing.
		diffsA := diffMainBytes(text1A, text2A, deadline)
		diffsB := diffMainBytes(text1B, text2B, deadline)
		// Merge the results.
		diffs := diffsA
		diffs = append(diffs, diff{OpCopy, clone(midCommon)})
		diffs = append(diffs, diffsB...)
		return diffs
		//} else if checklines && len(text1) > 100 && len(text2) > 100 {
		//	return dmp.diffLineMode(text1, text2, deadline)
	}
	return diffBisect(text1, text2, deadline)
}

// diffBisect finds the 'middle snake' of a diff, splits the problem in two and returns the recursively constructed diff.
// See Myers's 1986 paper: An O(ND) Difference Algorithm and Its Variations.
func diffBisect(runes1, runes2 []byte, deadline time.Time) []diff {
	// Cache the text lengths to prevent multiple calls.
	runes1Len, runes2Len := len(runes1), len(runes2)

	maxD := (runes1Len + runes2Len + 1) / 2
	vOffset := maxD
	vLength := 2 * maxD

	v1 := make([]int, vLength)
	v2 := make([]int, vLength)
	for i := range v1 {
		v1[i] = -1
		v2[i] = -1
	}
	v1[vOffset+1] = 0
	v2[vOffset+1] = 0

	delta := runes1Len - runes2Len
	// If the total number of characters is odd, then the front path will collide with the reverse path.
	front := (delta%2 != 0)
	// Offsets for start and end of k loop. Prevents mapping of space beyond the grid.
	k1start := 0
	k1end := 0
	k2start := 0
	k2end := 0
	for d := 0; d < maxD; d++ {
		// Bail out if deadline is reached.
		if !deadline.IsZero() && d%16 == 0 && time.Now().After(deadline) {
			break
		}

		// Walk the front path one step.
		for k1 := -d + k1start; k1 <= d-k1end; k1 += 2 {
			k1Offset := vOffset + k1
			var x1 int

			if k1 == -d || (k1 != d && v1[k1Offset-1] < v1[k1Offset+1]) {
				x1 = v1[k1Offset+1]
			} else {
				x1 = v1[k1Offset-1] + 1
			}

			y1 := x1 - k1
			for x1 < runes1Len && y1 < runes2Len {
				if runes1[x1] != runes2[y1] {
					break
				}
				x1++
				y1++
			}
			v1[k1Offset] = x1
			if x1 > runes1Len {
				// Ran off the right of the graph.
				k1end += 2
			} else if y1 > runes2Len {
				// Ran off the bottom of the graph.
				k1start += 2
			} else if front {
				k2Offset := vOffset + delta - k1
				if k2Offset >= 0 && k2Offset < vLength && v2[k2Offset] != -1 {
					// Mirror x2 onto top-left coordinate system.
					x2 := runes1Len - v2[k2Offset]
					if x1 >= x2 {
						// Overlap detected.
						return diffBisectSplit(runes1, runes2, x1, y1, deadline)
					}
				}
			}
		}
		// Walk the reverse path one step.
		for k2 := -d + k2start; k2 <= d-k2end; k2 += 2 {
			k2Offset := vOffset + k2
			var x2 int
			if k2 == -d || (k2 != d && v2[k2Offset-1] < v2[k2Offset+1]) {
				x2 = v2[k2Offset+1]
			} else {
				x2 = v2[k2Offset-1] + 1
			}
			var y2 = x2 - k2
			for x2 < runes1Len && y2 < runes2Len {
				if runes1[runes1Len-x2-1] != runes2[runes2Len-y2-1] {
					break
				}
				x2++
				y2++
			}
			v2[k2Offset] = x2
			if x2 > runes1Len {
				// Ran off the left of the graph.
				k2end += 2
			} else if y2 > runes2Len {
				// Ran off the top of the graph.
				k2start += 2
			} else if !front {
				k1Offset := vOffset + delta - k2
				if k1Offset >= 0 && k1Offset < vLength && v1[k1Offset] != -1 {
					x1 := v1[k1Offset]
					y1 := vOffset + x1 - k1Offset
					// Mirror x2 onto top-left coordinate system.
					x2 = runes1Len - x2
					if x1 >= x2 {
						// Overlap detected.
						return diffBisectSplit(runes1, runes2, x1, y1, deadline)
					}
				}
			}
		}
	}
	// diff took too long and hit the deadline or number of diffs equals number of characters, no commonality at all.
	return []diff{
		diff{OpDelete, clone(runes1)},
		diff{OpInsert, clone(runes2)},
	}
}

func diffBisectSplit(runes1, runes2 []byte, x, y int,
	deadline time.Time) []diff {
	runes1a := runes1[:x]
	runes2a := runes2[:y]
	runes1b := runes1[x:]
	runes2b := runes2[y:]

	// Compute both diffs serially.
	diffs := diffMainBytes(runes1a, runes2a, deadline)
	diffsb := diffMainBytes(runes1b, runes2b, deadline)

	return append(diffs, diffsb...)
}

// commonPrefixLength returns the length of the common prefix of two rune slices.
func commonPrefixLength(text1, text2 []byte) int {
	// Linear search. See comment in commonSuffixLength.
	n := 0
	for ; n < len(text1) && n < len(text2); n++ {
		if text1[n] != text2[n] {
			return n
		}
	}
	return n
}

// commonSuffixLength returns the length of the common suffix of two rune slices.
func commonSuffixLength(text1, text2 []byte) int {
	// Use linear search rather than the binary search discussed at https://neil.fraser.name/news/2007/10/09/.
	// See discussion at https://github.com/sergi/go-diff/issues/54.
	i1 := len(text1)
	i2 := len(text2)
	for n := 0; ; n++ {
		i1--
		i2--
		if i1 < 0 || i2 < 0 || text1[i1] != text2[i2] {
			return n
		}
	}
}

// diffCommonOverlap determines if the suffix of one string is the prefix of another.
func diffCommonOverlap(text1 []byte, text2 []byte) int {
	// Cache the text lengths to prevent multiple calls.
	text1Length := len(text1)
	text2Length := len(text2)
	// Eliminate the null case.
	if text1Length == 0 || text2Length == 0 {
		return 0
	}
	// Truncate the longer string.
	if text1Length > text2Length {
		text1 = text1[text1Length-text2Length:]
	} else if text1Length < text2Length {
		text2 = text2[0:text1Length]
	}
	textLength := int(math.Min(float64(text1Length), float64(text2Length)))
	// Quick check for the worst case.
	if bytes.Equal(text1, text2) {
		return textLength
	}

	// Start by looking for a single character match and increase length until no match is found. Performance analysis: http://neil.fraser.name/news/2010/11/04/
	best := 0
	length := 1
	for {
		pattern := text1[textLength-length:]
		found := bytes.Index(text2, pattern)
		if found == -1 {
			break
		}
		length += found
		if found == 0 || bytes.Equal(text1[textLength-length:], text2[0:length]) {
			best = length
			length++
		}
	}

	return best
}

func diffHalfMatch(text1, text2 []byte, unlimitedTime bool) [][]byte {
	if unlimitedTime {
		// Don't risk returning a non-optimal diff if we have unlimited time.
		return nil
	}

	var longtext, shorttext []byte
	if len(text1) > len(text2) {
		longtext = text1
		shorttext = text2
	} else {
		longtext = text2
		shorttext = text1
	}

	if len(longtext) < 4 || len(shorttext)*2 < len(longtext) {
		return nil // Pointless.
	}

	// First check if the second quarter is the seed for a half-match.
	hm1 := diffHalfMatchI(longtext, shorttext, int(float64(len(longtext)+3)/4))

	// Check again based on the third quarter.
	hm2 := diffHalfMatchI(longtext, shorttext, int(float64(len(longtext)+1)/2))

	hm := [][]byte{}
	if hm1 == nil && hm2 == nil {
		return nil
	} else if hm2 == nil {
		hm = hm1
	} else if hm1 == nil {
		hm = hm2
	} else {
		// Both matched.  Select the longest.
		if len(hm1[4]) > len(hm2[4]) {
			hm = hm1
		} else {
			hm = hm2
		}
	}

	// A half-match was found, sort out the return data.
	if len(text1) > len(text2) {
		return hm
	}

	return [][]byte{hm[2], hm[3], hm[0], hm[1], hm[4]}
}

// diffHalfMatchI checks if a substring of shorttext exist within longtext such that the substring is at least half the length of longtext?
// Returns a slice containing the prefix of longtext, the suffix of longtext, the prefix of shorttext, the suffix of shorttext and the common middle, or null if there was no match.
func diffHalfMatchI(l, s []byte, i int) [][]byte {
	var bestCommonA []byte
	var bestCommonB []byte
	var bestCommonLen int
	var bestLongtextA []byte
	var bestLongtextB []byte
	var bestShorttextA []byte
	var bestShorttextB []byte

	// Start with a 1/4 length substring at position i as a seed.
	seed := l[i : i+len(l)/4]

	for j := bytesIndexOf(s, seed, 0); j != -1; j = bytesIndexOf(s, seed, j+1) {
		prefixLength := commonPrefixLength(l[i:], s[j:])
		suffixLength := commonSuffixLength(l[:i], s[:j])

		if bestCommonLen < suffixLength+prefixLength {
			bestCommonA = s[j-suffixLength : j]
			bestCommonB = s[j : j+prefixLength]
			bestCommonLen = len(bestCommonA) + len(bestCommonB)
			bestLongtextA = l[:i-suffixLength]
			bestLongtextB = l[i+prefixLength:]
			bestShorttextA = s[:j-suffixLength]
			bestShorttextB = s[j+prefixLength:]
		}
	}

	if bestCommonLen*2 < len(l) {
		return nil
	}

	return [][]byte{
		bestLongtextA,
		bestLongtextB,
		bestShorttextA,
		bestShorttextB,
		append(bestCommonA, bestCommonB...),
	}
}

// diffCleanupSemantic reduces the number of edits by eliminating semantically trivial equalities.
func diffCleanupSemantic(diffs []diff) []diff {
	changes := false
	// Stack of indices where equalities are found.
	equalities := make([]int, 0, len(diffs))

	var lastequality []byte
	// Always equal to diffs[equalities[equalitiesLength - 1]][1]
	var pointer int // Index of current position.
	// Number of characters that changed prior to the equality.
	var lengthInsertions1, lengthDeletions1 int
	// Number of characters that changed after the equality.
	var lengthInsertions2, lengthDeletions2 int

	for pointer < len(diffs) {
		if diffs[pointer].Type == OpCopy {
			// Equality found.
			equalities = append(equalities, pointer)
			lengthInsertions1 = lengthInsertions2
			lengthDeletions1 = lengthDeletions2
			lengthInsertions2 = 0
			lengthDeletions2 = 0
			lastequality = clone(diffs[pointer].Text)
		} else {
			// An insertion or deletion.

			if diffs[pointer].Type == OpInsert {
				lengthInsertions2 += len(diffs[pointer].Text)
			} else {
				lengthDeletions2 += len(diffs[pointer].Text)
			}
			// Eliminate an equality that is smaller or equal to the edits on both sides of it.
			difference1 := int(math.Max(float64(lengthInsertions1), float64(lengthDeletions1)))
			difference2 := int(math.Max(float64(lengthInsertions2), float64(lengthDeletions2)))
			if len(lastequality) > 0 &&
				(len(lastequality) <= difference1) &&
				(len(lastequality) <= difference2) {
				// Duplicate record.
				insPoint := equalities[len(equalities)-1]
				diffs = splice(diffs, insPoint, 0, diff{OpDelete, lastequality})

				// Change second copy to insert.
				diffs[insPoint+1].Type = OpInsert
				// Throw away the equality we just deleted.
				equalities = equalities[:len(equalities)-1]

				if len(equalities) > 0 {
					equalities = equalities[:len(equalities)-1]
				}
				pointer = -1
				if len(equalities) > 0 {
					pointer = equalities[len(equalities)-1]
				}

				lengthInsertions1 = 0 // Reset the counters.
				lengthDeletions1 = 0
				lengthInsertions2 = 0
				lengthDeletions2 = 0
				lastequality = nil
				changes = true
			}
		}
		pointer++
	}

	// Normalize the diff.
	if changes {
		diffs = diffCleanupMerge(diffs)
	}
	// diffs = dmp.DiffCleanupSemanticLossless(diffs)
	// Find any overlaps between deletions and insertions.
	// e.g: <del>abcxxx</del><ins>xxxdef</ins>
	//   -> <del>abc</del>xxx<ins>def</ins>
	// e.g: <del>xxxabc</del><ins>defxxx</ins>
	//   -> <ins>def</ins>xxx<del>abc</del>
	// Only extract an overlap if it is as big as the edit ahead or behind it.
	pointer = 1
	for pointer < len(diffs) {
		if diffs[pointer-1].Type == OpDelete &&
			diffs[pointer].Type == OpInsert {
			deletion := diffs[pointer-1].Text
			insertion := diffs[pointer].Text
			overlapLength1 := diffCommonOverlap(deletion, insertion)
			overlapLength2 := diffCommonOverlap(insertion, deletion)
			if overlapLength1 >= overlapLength2 {
				if float64(overlapLength1) >= float64(len(deletion))/2 ||
					float64(overlapLength1) >= float64(len(insertion))/2 {

					// Overlap found. Insert an equality and trim the surrounding edits.
					diffs = splice(diffs, pointer, 0, diff{OpCopy, insertion[:overlapLength1]})
					diffs[pointer-1].Text =
						deletion[0 : len(deletion)-overlapLength1]
					diffs[pointer+1].Text = insertion[overlapLength1:]
					pointer++
				}
			} else {
				if float64(overlapLength2) >= float64(len(deletion))/2 ||
					float64(overlapLength2) >= float64(len(insertion))/2 {
					// Reverse overlap found. Insert an equality and swap and trim the surrounding edits.
					overlap := diff{OpCopy, deletion[:overlapLength2]}
					diffs = splice(diffs, pointer, 0, overlap)
					diffs[pointer-1].Type = OpInsert
					diffs[pointer-1].Text = insertion[0 : len(insertion)-overlapLength2]
					diffs[pointer+1].Type = OpDelete
					diffs[pointer+1].Text = deletion[overlapLength2:]
					pointer++
				}
			}
			pointer++
		}
		pointer++
	}

	return diffs
}

// diffCleanupEfficiency reduces the number of edits by eliminating operationally trivial equalities.
func diffCleanupEfficiency(diffs []diff) []diff {
	changes := false
	// Stack of indices where equalities are found.
	type equality struct {
		data int
		next *equality
	}
	var equalities *equality
	// Always equal to equalities[equalitiesLength-1][1]
	var lastequality []byte
	pointer := 0 // Index of current position.
	// Is there an insertion operation before the last equality.
	preIns := false
	// Is there a deletion operation before the last equality.
	preDel := false
	// Is there an insertion operation after the last equality.
	postIns := false
	// Is there a deletion operation after the last equality.
	postDel := false
	for pointer < len(diffs) {
		if diffs[pointer].Type == OpCopy { // Equality found.
			if len(diffs[pointer].Text) < diffEditCost &&
				(postIns || postDel) {
				// Candidate found.
				equalities = &equality{
					data: pointer,
					next: equalities,
				}
				preIns = postIns
				preDel = postDel
				lastequality = clone(diffs[pointer].Text)
			} else {
				// Not a candidate, and can never become one.
				equalities = nil
				lastequality = nil
			}
			postIns = false
			postDel = false
		} else { // An insertion or deletion.
			if diffs[pointer].Type == OpDelete {
				postDel = true
			} else {
				postIns = true
			}

			// Five types to be split:
			// <ins>A</ins><del>B</del>XY<ins>C</ins><del>D</del>
			// <ins>A</ins>X<ins>C</ins><del>D</del>
			// <ins>A</ins><del>B</del>X<ins>C</ins>
			// <ins>A</del>X<ins>C</ins><del>D</del>
			// <ins>A</ins><del>B</del>X<del>C</del>
			var sumPres int
			if preIns {
				sumPres++
			}
			if preDel {
				sumPres++
			}
			if postIns {
				sumPres++
			}
			if postDel {
				sumPres++
			}
			if len(lastequality) > 0 &&
				((preIns && preDel && postIns && postDel) ||
					((len(lastequality) < diffEditCost/2) && sumPres == 3)) {

				insPoint := equalities.data

				// Duplicate record.
				diffs = splice(diffs, insPoint, 0, diff{OpDelete, lastequality})

				// Change second copy to insert.
				diffs[insPoint+1].Type = OpInsert
				// Throw away the equality we just deleted.
				equalities = equalities.next
				lastequality = nil

				if preIns && preDel {
					// No changes made which could affect previous entry, keep going.
					postIns = true
					postDel = true
					equalities = nil
				} else {
					if equalities != nil {
						equalities = equalities.next
					}
					if equalities != nil {
						pointer = equalities.data
					} else {
						pointer = -1
					}
					postIns = false
					postDel = false
				}
				changes = true
			}
		}
		pointer++
	}

	if changes {
		diffs = diffCleanupMerge(diffs)
	}

	return diffs
}

// diffCleanupMerge reorders and merges like edit sections. Merge equalities.
// Any edit section can move as long as it doesn't cross an equality.
func diffCleanupMerge(diffs []diff) []diff {
	// Add a dummy entry at the end.
	diffs = append(diffs, diff{OpCopy, nil})
	pointer := 0
	countDelete := 0
	countInsert := 0
	commonlength := 0
	var textDelete []byte
	var textInsert []byte

	for pointer < len(diffs) {
		switch diffs[pointer].Type {
		case OpInsert:
			countInsert++
			textInsert = append(textInsert, diffs[pointer].Text...)
			pointer++
			break
		case OpDelete:
			countDelete++
			textDelete = append(textDelete, diffs[pointer].Text...)
			pointer++
			break
		case OpCopy:
			// Upon reaching an equality, check for prior redundancies.
			if countDelete+countInsert > 1 {
				if countDelete != 0 && countInsert != 0 {
					// Factor out any common prefixies.
					commonlength = commonPrefixLength(textInsert, textDelete)
					if commonlength != 0 {
						x := pointer - countDelete - countInsert
						if x > 0 && diffs[x-1].Type == OpCopy {
							diffs[x-1].Text = append(diffs[x-1].Text, textInsert[:commonlength]...)
						} else {
							diffs = append([]diff{diff{OpCopy, clone(textInsert[:commonlength])}}, diffs...)
							pointer++
						}
						textInsert = textInsert[commonlength:]
						textDelete = textDelete[commonlength:]
					}
					// Factor out any common suffixies.
					commonlength = commonSuffixLength(textInsert, textDelete)
					if commonlength != 0 {
						insertIndex := len(textInsert) - commonlength
						deleteIndex := len(textDelete) - commonlength
						diffs[pointer].Text = cleanAppend(textInsert[insertIndex:], diffs[pointer].Text)
						textInsert = textInsert[:insertIndex]
						textDelete = textDelete[:deleteIndex]
					}
				}
				// Delete the offending records and add the merged ones.
				if countDelete == 0 {
					diffs = splice(diffs, pointer-countInsert,
						countDelete+countInsert,
						diff{OpInsert, clone(textInsert)})
				} else if countInsert == 0 {
					diffs = splice(diffs, pointer-countDelete,
						countDelete+countInsert,
						diff{OpDelete, clone(textDelete)})
				} else {
					diffs = splice(diffs, pointer-countDelete-countInsert,
						countDelete+countInsert,
						diff{OpDelete, clone(textDelete)},
						diff{OpInsert, clone(textInsert)})
				}

				pointer = pointer - countDelete - countInsert + 1
				if countDelete != 0 {
					pointer++
				}
				if countInsert != 0 {
					pointer++
				}
			} else if pointer != 0 && diffs[pointer-1].Type == OpCopy {
				// Merge this equality with the previous one.
				diffs[pointer-1].Text = cleanAppend(diffs[pointer-1].Text, diffs[pointer].Text)
				diffs = append(diffs[:pointer], diffs[pointer+1:]...)
			} else {
				pointer++
			}
			countInsert = 0
			countDelete = 0
			textDelete = nil
			textInsert = nil
			break
		}
	}

	if len(diffs[len(diffs)-1].Text) == 0 {
		diffs = diffs[0 : len(diffs)-1] // Remove the dummy entry at the end.
	}

	// Second pass: look for single edits surrounded on both sides by equalities which can be shifted sideways to eliminate an equality. E.g: A<ins>BA</ins>C -> <ins>AB</ins>AC
	changes := false
	pointer = 1
	// Intentionally ignore the first and last element (don't need checking).
	for pointer < (len(diffs) - 1) {
		if diffs[pointer-1].Type == OpCopy &&
			diffs[pointer+1].Type == OpCopy {
			// This is a single edit surrounded by equalities.
			if bytes.HasSuffix(diffs[pointer].Text, diffs[pointer-1].Text) {
				// Shift the edit over the previous equality.
				diffs[pointer].Text = cleanAppend(diffs[pointer-1].Text,
					diffs[pointer].Text[:len(diffs[pointer].Text)-len(diffs[pointer-1].Text)])
				diffs[pointer+1].Text = cleanAppend(diffs[pointer-1].Text, diffs[pointer+1].Text)
				diffs = splice(diffs, pointer-1, 1)
				changes = true
			} else if bytes.HasPrefix(diffs[pointer].Text, diffs[pointer+1].Text) {
				// Shift the edit over the next equality.
				diffs[pointer-1].Text = cleanAppend(diffs[pointer-1].Text, diffs[pointer+1].Text)
				diffs[pointer].Text =
					cleanAppend(diffs[pointer].Text[len(diffs[pointer+1].Text):], diffs[pointer+1].Text)
				diffs = splice(diffs, pointer+1, 1)
				changes = true
			}
		}
		pointer++
	}

	// If shifts were made, the diff needs reordering and another shift sweep.
	if changes {
		diffs = diffCleanupMerge(diffs)
	}

	return diffs
}

// splice removes amount elements from slice at index index, replacing them with elements.
func splice(slice []diff, index int, amount int, elements ...diff) []diff {
	if len(elements) == amount {
		// Easy case: overwrite the relevant items.
		copy(slice[index:], elements)
		return slice
	}
	if len(elements) < amount {
		// Fewer new items than old.
		// Copy in the new items.
		copy(slice[index:], elements)
		// Shift the remaining items left.
		copy(slice[index+len(elements):], slice[index+amount:])
		// Calculate the new end of the slice.
		end := len(slice) - amount + len(elements)
		// Zero stranded elements at end so that they can be garbage collected.
		tail := slice[end:]
		for i := range tail {
			tail[i] = diff{}
		}
		return slice[:end]
	}
	// More new items than old.
	// Make room in slice for new elements.
	// There's probably an even more efficient way to do this,
	// but this is simple and clear.
	need := len(slice) - amount + len(elements)
	for len(slice) < need {
		slice = append(slice, diff{})
	}
	// Shift slice elements right to make room for new elements.
	copy(slice[index+len(elements):], slice[index+amount:])
	// Copy in new elements.
	copy(slice[index:], elements)
	return slice
}

// clone makes a copy of a byte slice. The original go-diff library used strings
// throughout, which are immutable. Subtle slice aliasing bug following a straight
// string->[]byte conversion prompted lots of cloning.
func clone(in []byte) []byte {
	out := make([]byte, len(in))
	copy(out, in)
	return out
}

// cleanAppend concatenates multiple byte slices into a single slice
// while leaving the originals untouched.
func cleanAppend(slices ...[]byte) []byte {
	cap := 0
	for _, s := range slices {
		cap += len(s)
	}

	ret := make([]byte, 0, cap)
	for _, s := range slices {
		ret = append(ret, s...)
	}
	return ret
}

func bytesIndexOf(target, pattern []byte, i int) int {
	if i > len(target)-2 {
		return -1
	}

	ind := bytes.Index(target[i:], pattern)
	if ind == -1 {
		return -1
	}
	return ind + i
}
