package sudoku_solver

/* This solver is based on an algorithm published by David Eppstein in his PADS
 * package at https://www.ics.uci.edu/~eppstein/PADS/ and in particular
 * https://www.ics.uci.edu/~eppstein/PADS/Sudoku.py
 *
 * This algorithm uses a 81 bit long Python int to represent all possible
 * candidates living in a cell.
 * These days a uint128 can be used for doing this sort of thing as well
 *
 * I am utilizing the unit128 package from lukechampine.com/uint128 at
 * https://github.com/lukechampine/uint128.
 * I have removed a few methods which I don't need (multipication, division)
 * and added a few to make life easier for me (ToOctal and ToHex for debugging),
 * Not() and Less for the proper functioning of the solving algorithm
 *
 * The input format for the puzzles to solve is 81 characters of 0..9 or '.'
 * if the line is longer than 81 characers, it gets trimmed to size.
 */

import (
  "fmt"
  "strings"

  // local
  "github.com/wplapper/go-sudoku3/uint128"
  "github.com/wplapper/go-sudoku3/sudoku_constants"
)


// global variables - makes life easier
// quasi constants
const Nine = 9
var DEBUG = 0
var ONE = uint128.From64(1)
var ALL_ONE = uint128.From64(1).Lsh(81).Sub(ONE)

// dynamic variables
var locations[10] uint128.Uint128
var contents[81] int
// 'unit_solved' is 2 dimensional slice containing true / false for the
// combinations of all digits (1..9) for all groups(9+9+9)
var unit_solved [][] bool
var progress bool

// pointer arrays
var p_all_powers[]          *uint128.Uint128
var p_Alignments_bysqua[][] *uint128.Uint128 //  9 * 24
var p_Alignments_byline[][] *uint128.Uint128 // 18 * 12

func Setup_solver_once() {
    sudoku_constants.Setup_sudoku_constants()
    // setup unit_solved
    unit_solved = make([][]bool, 10)
    for i := range unit_solved {
        unit_solved[i] = make([] bool, 27)
    }

    //setup pointer list to sudoku_constants.Powers, needed for bisect
    p_all_powers = make([] *uint128.Uint128, 81)
    for pos := 0; pos < len(sudoku_constants.Powers); pos++ {
        p_all_powers[pos] = &sudoku_constants.Powers[pos]
    }

    // setup pointer array for 'p_Alignments_bysqua'
    p_Alignments_bysqua = make([][] *uint128.Uint128, 9)
    for i := 0; i < len(sudoku_constants.Alignments_bysqua); i++ {
        p_Alignments_bysqua[i] = make([] *uint128.Uint128, 24)
    }

    // fill 'p_Alignments_bysqua'
    for outer := 0; outer < 9; outer++ {
        for inner := 0; inner < 24; inner++ {
            p_Alignments_bysqua[outer][inner] =
                &sudoku_constants.Alignments_bysqua[outer][inner].Mask
        }
    }

    // setup pointer array for 'p_Alignments_byline'
    p_Alignments_byline = make([][] *uint128.Uint128, 18)
    for i := 0; i < len(sudoku_constants.Alignments_byline); i++ {
        p_Alignments_byline[i] = make([] *uint128.Uint128, 12)
    }

    // fill 'p_Alignments_byline'
    for outer := 0; outer < 18; outer++ {
        for inner := 0; inner < 12; inner++ {
            p_Alignments_byline[outer][inner] =
                &sudoku_constants.Alignments_byline[outer][inner].Mask
        }
    }
}

func Start_solver(puzzle string) int {
    // reset contents, locations and unit_solved,
    // load initial values into locations etc.
    var digit int

    // reset locations to all possible candidates
    for d := 1; d <= Nine; d++ {
        locations[d] = ALL_ONE
    }

    // reset contents to zero
    for cell := 0; cell < Nine * Nine;  cell++ {
        contents[cell] = 0
    }

    // reset unit_solved to false
    for d := 1; d <= Nine; d++ {
        for g := 0; g < Nine * 3; g++ {
            unit_solved[d][g] = false
        }
    }

    // fill known places from puzzle
    length := len(puzzle)
    if  length < 81 {
        panic("puzzle length not 81")
    } else if length > 81 {
        puzzle = puzzle[:81]
    }

    for cell, char := range puzzle {
        if '0' <= char && char <= '9' {
            digit = int(char) - 48 // 48 == '0'
            place(digit, cell, sudoku_constants.Powers[cell])
        }
    }

    Solve()
    return count_content()
}

func Solve() int {
    // call the defined solver functions in sequence
    // if a solver succeeds, restart from the beginning
    var count int
    var res bool

    // need a type declaration for function pointers
    type SolveFunc func() bool
    funcname  := [3] string {"locate", "single", "align"}
    functions := [3] SolveFunc {locate, single, align}

    progress = true
    for progress {
        count = count_content()
        if count == 81 {
            return count
        }

        for pos, function := range functions {
            if DEBUG > 0 {
                fmt.Printf("calling function %s\n", funcname[pos])
            }
            res = function()
            if res {
                // if successful , restart from the beginning
                break
            }
        } // oop over solver functions
    } // end while

    // OnesCount
    return count_content()
}
/*==============================================================================
 *  solver functions
 *==============================================================================
 */
func locate() bool {
    // find digits which only live in one place in a group
    var mask uint128.Uint128
    var cell int

    progress = false
    for d := 1; d <= Nine; d++ {
        for g := 0; g < Nine * 3; g++ {
            if unit_solved[d][g] {
                continue
            }

            mask = locations[d].And(sudoku_constants.Group_masks[g])
            if ! (mask.And(mask.Sub(ONE))).IsZero() {
                continue
            }

            // found a single occupant for digit 'd' in group 'g'
            cell = bisect(mask, p_all_powers)
            if cell < 0 {
                fmt.Printf("impossible\n")
                panic("bisect error")
            }

            if DEBUG > 0 {
                fmt.Printf("place with d=%d g=%2d for cell %s\n",
                    d, g, lin2name(cell))
            }
            place(d, cell, mask)
        }
    }
    return progress
}

func single() bool {
    // find cell which one have one candidate left in the 'cell'
    var count, dd int
    var bit uint128.Uint128

    progress = false
    for cell := 0; cell < Nine * Nine; cell++ {
        if contents[cell] != 0 {
            continue
        }

        count = 0
        bit = sudoku_constants.Powers[cell]
        for d := 1; d <= Nine; d++ {
            if ! (locations[d].And(bit)).IsZero() {
                count++
                dd = d
                if count > 1 {
                    break
                }
            }
        }

        if count == 1 {
            if DEBUG > 0 {
                fmt.Printf("single %d in %s\n", dd, lin2name(cell))
            }
            // found at single candidate 'dd' at 'cell'
            place(dd, cell, bit)
        }
    }
    return progress
}

func align() bool {
    // check for candidates which live only in one row/column
    var mask, m uint128.Uint128
    var sm, c  int

    progress = false
    for d := 1; d <= Nine; d++ {
        //try the columns / rows first
        for g := Nine; g < Nine * 3; g++ {
            if unit_solved[d][g] {
                continue
            }
            mask = locations[d].And(sudoku_constants.Group_masks[g])
            c    = bisect(mask, p_Alignments_byline[g - 9])
            if c < 0 {
                continue
            }
            sm = sudoku_constants.Alignments_byline[g - 9][c].S
            m  = sudoku_constants.Group_masks[sm].And(locations[d]).
                And(mask.Not())
            if m.IsZero() {
                continue
            }
            if DEBUG > 0 {
                locs := mask2cellnames(m)
                fmt.Printf("align1 d=%d at %2d X %2d for locs %s\n",
                    d, g, sm, strings.Join(locs, ","))
            }
            unplace(d, m)
        }
    }

    // go along and try the boxes
    for d := 1; d <= Nine; d++ {
        for s := 0; s < Nine; s++ {
            if unit_solved[d][s] {
                continue
            }
            mask = locations[d].And(sudoku_constants.Group_masks[s])
            c    = bisect(mask, p_Alignments_bysqua[s])
            if c < 0 {
                continue
            }
            sm = sudoku_constants.Alignments_bysqua[s][c].G
            m  = sudoku_constants.Group_masks[sm].And(locations[d]).
                And(mask.Not())
            if m.IsZero() {
                continue
            }
            if DEBUG > 0 {
                locs := mask2cellnames(m)
                fmt.Printf("align2 d=%d at %2d X %2d for locs %v\n",
                    d, s, sm, strings.Join(locs, ","))
            }
            unplace(d, m)
        }
    }
    return progress
}

/*==============================================================================
 *  helpers for solver functions: place and unplace
 *==============================================================================
 */
func place(digit int, cell int, bit uint128.Uint128) bool {
    // put digit 'digit' into 'cell'
    contents[cell] = digit
    not_bit := bit.Not()
    var value [] int

    // remove candidates which have been fixed
    for d := 1; d <= Nine; d++ {
        if d != digit {
            locations[d] = locations[d].And(not_bit)
        } else {
            locations[d] = locations[d].And(
                sudoku_constants.Neighbours[cell].Not())
        }
    }

    // set unit_solved
    value = sudoku_constants.Unit_index[cell]
    unit_solved[digit][value[0]] = true
    unit_solved[digit][value[1]] = true
    unit_solved[digit][value[2]] = true
    progress = true
    return progress
}

func unplace(digit int, mask uint128.Uint128) bool {
    // remove candidates from puzzle
    if ! locations[digit].And(mask).IsZero() {
        locations[digit] = locations[digit].And(mask.Not())
        progress = true
    }
    return progress
}

/*==============================================================================
 *  utility functions
 *==============================================================================
 */
func bisect(mask uint128.Uint128, all_masks []*uint128.Uint128) int {
    // bisect all_masks and find match point

    var ptr *uint128.Uint128
    var print_out string
    high := len(all_masks) - 1
    if DEBUG > 2 && high != 80 {
        print_out = fmt.Sprintf("\nbis high=%2d mask='%s'\n", high,
            mask.ToOctal())
    }

    var mid int
    low := 0
    for low <= high {
        mid = (low + high) >> 1
        if DEBUG > 5 {
            fmt.Printf("lo=%2d mi=%2d hi=%2d\n", low, mid, high)
        }
        ptr = all_masks[mid]
        if mask.Equals(*ptr) {
            if DEBUG > 2 && len(print_out) > 0 {
                fmt.Printf("%s", print_out)
            }
            return mid
        } else if mask.Less(*ptr) {
            high = mid - 1
        } else {
            low = mid + 1
        }
    }
    return -1
}

func lin2name(cell int) string {
    // convert linear 'cell' to a two-dimensional sudoku address
    return fmt.Sprintf("[%d%d]", cell / 9 + 1, cell % 9 + 1)
}

func count_content() int {
    // count the number of soved cells in the current puzzle
    var count = 0
    for cell := 0; cell < Nine * Nine; cell++ {
        if contents[cell] > 0 {
            count++
        }
    }
    return count
}

func mask2cellnames(mask uint128.Uint128) []string {
    // convert canddates into locations
    var locs [] string

    for ! mask.IsZero() {
        bit := mask.And((mask.Sub(ONE)).Not())
        mask = mask.And(bit.Not())
        cell := bisect(bit, p_all_powers)
        locs = append(locs, lin2name(cell))
    }
    return locs
}
