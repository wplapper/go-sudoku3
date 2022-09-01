package sudoku_constants

/* This solver is based on an algorithm published by David Eppstein in his PADS
 * package at https://www.ics.uci.edu/~eppstein/PADS/ and in particular
 * https://www.ics.uci.edu/~eppstein/PADS/Sudoku.py
 *
 * The uint128 implementation from https://github.com/lukechampine/uint128
 */

import (
  "sort"

  // local
  "github.com/wplapper/go-sudoku3/uint128"
)

const Nine = 9
type Pylong uint128.Uint128

var Powers[]      uint128.Uint128
var Group_masks[] uint128.Uint128
var Neighbours[]  uint128.Uint128

var ONE = uint128.From64(1)

type Align struct{
    // need to Capitaize member names, so that they are visible here and outside
    Mask uint128.Uint128
    S, G int
}

// Alignments_bysqua and Alignments_bysqua are accelerators for searching the
// alignments structures with 'bisect'.
// Instead of bisecting one large 216 alignments slice,
// it is faster to bisect 9 or 18 smaller ones for a small size increase
// of the execution space.
var Alignments_bysqua[][] Align //  9 * 24
var Alignments_byline[][] Align // 18 * 12
var Unit_index [][] int

// basic building blocks
func setup_powers() {
    // prepare all 2**i
    Powers = make([] uint128.Uint128, 81)
    temp := uint128.From64(1)
    for i := 0; i < Nine * Nine; i++ {
        Powers[i] = temp
        temp = temp.Lsh(1)
    }
}

// each cell lives in three spaces simultaneously, in a row (r), in a column (c)
// and in a square = box (b). 'Unit_index' memorizes these space numbers.
func setup_unit_index() {
    // setup 'unit_index[81][3]'
    Unit_index = make([][]int, 81)
    for i := range Unit_index {
        Unit_index[i] = make([] int, 3)
    }

    // fill in values for 'unit_index'
    var r, c, b int
    for cell := 0; cell < Nine * Nine; cell++ {
        r = cell / 9
        c = cell % 9
        b = r / 3 * 3 + c / 3
        Unit_index[cell][0] = b
        Unit_index[cell][1] = r +  9
        Unit_index[cell][2] = c + 18
    }
}

func setup_group_masks() {
    // we will define a grop mask for boxes (squares), rows ans=d columns
    var indbox int
    Group_masks = make([] uint128.Uint128, 27)
    for r := 0; r < Nine; r++ {
        for c := 0; c < Nine; c++ {
            indbox = ((r / 3 * 9 + r % 3) * 3) + (c / 3 * 9 + c % 3)
            Group_masks[r   ] = Group_masks[r   ].Or(Powers[indbox])
            Group_masks[r+ 9] = Group_masks[r+ 9].Or(Powers[r * 9 + c])
            Group_masks[r+18] = Group_masks[r+18].Or(Powers[c * 9 + r])
        }
    }
}

func setup_neighbours() {
    // the neigbours are needed when fixing a candidate in a cell
    var r, c, b int
    var value [] int
    Neighbours = make([] uint128.Uint128, 81)
    for cell := 0; cell < Nine * Nine; cell++ {
        value = Unit_index[cell]
        b = value[0]
        r = value[1]
        c = value[2]
        Neighbours[cell] = (Group_masks[r].Or(Group_masks[c]).
            Or(Group_masks[b])).Xor(Powers[cell])
    }
}

func setup_alignments_2dim() {
    // split up all Alignments (216) into 9 * 24 and 18 *12 elements
    Alignments_bysqua = make([][] Align, 9)
    for i := range Alignments_bysqua {
        Alignments_bysqua[i] = make([] Align, 24)
    }

    Alignments_byline = make([][] Align, 18)
    for i := range Alignments_byline {
        Alignments_byline[i] = make([] Align, 12)
    }
}

func setup_alignments() {
    // intersect all boxes with all rows and columns

    // temps
    var b1, b2, b3, original_mask uint128.Uint128
    count := 0
    alignments := make([] Align, 216)

    // define comparison function for sort
    cmp_align_s_cmp := func (i, j int) bool {
        if alignments[i].S < alignments[j].S {
            return true
        } else if alignments[i].S > alignments[j].S {
            return false
        } else {
            return alignments[i].Mask.Less(alignments[j].Mask)
        }
    }

    cmp_align_g_cmp := func (i, j int) bool {
        if alignments[i].G < alignments[j].G {
            return true
        } else if alignments[i].G > alignments[j].G {
            return false
        } else {
            return alignments[i].Mask.Less(alignments[j].Mask)
        }
    }

    // intersect all boxes
    for b := 0; b < Nine; b++ {
        // with all rows and columns
        for rc := Nine; rc < Nine * 3; rc++ {
            mask := Group_masks[b].And(Group_masks[rc])
            if mask.IsZero() {
              continue
            }

            // we will have 54 cases of intersections
            // set up an alignment block (4 values each for 3-bits and 3*2-bits)
            for ii := count; ii < count + 4; ii++ {
                alignments[ii].S = b
                alignments[ii].G = rc
            }

            // set the mask
            // b1, b2 and b3 are single bit masks
            original_mask = mask
            b1   = mask.And((mask.Sub(ONE)).Not())
            mask = mask.And(b1.Not())
            b2   = mask.And((mask.Sub(ONE)).Not())
            b3   = mask.And(b2.Not())

            // 3 bits in this mask
            alignments[count].Mask = original_mask

            // all posible 2 bit combinations of 'mask'
            alignments[count + 1].Mask = b1.Or(b2)
            alignments[count + 2].Mask = b2.Or(b3)
            alignments[count + 3].Mask = b3.Or(b1)
            count += 4
        }
    }

    // sort by square: 's' first, 'mask' second
    sort.SliceStable(alignments, cmp_align_s_cmp)

    // copy to Alignments_bysqua
    count = 0
    for outer := 0; outer < 9; outer++ {
        for inner := 0; inner < 24; inner++ {
          Alignments_bysqua[outer][inner] = alignments[count]
          count++
        }
    }

    // sort by row and column: 'g' first, 'mask' second
    sort.SliceStable(alignments, cmp_align_g_cmp)

    // copy to Alignments_byline
    count = 0
    for outer := 0; outer < 18; outer++ {
        for inner := 0; inner < 12; inner++ {
          Alignments_byline[outer][inner] = alignments[count]
          count++
        }
    }
}

func Setup_sudoku_constants() {
    setup_powers()
    setup_unit_index()
    setup_group_masks()
    setup_neighbours()
    setup_alignments_2dim()
    setup_alignments()
}
