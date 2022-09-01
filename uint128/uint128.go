package uint128 // import "lukechampine.com/uint128"

import (
  "encoding/binary"
  "errors"
  "fmt"
  "math"
  "math/big"
  "math/bits"
  "strings"
)

// Zero is a zero-valued uint128.
var Zero Uint128

// Max is the largest possible uint128 value.
var Max = New(math.MaxUint64, math.MaxUint64)

// A Uint128 is an unsigned 128-bit number.
type Uint128 struct {
  Lo, Hi uint64
}

// IsZero returns true if u == 0.
func (u Uint128) IsZero() bool {
  // NOTE: we do not compare against Zero, because that is a global variable
  // that could be modified.
  return u == Uint128{Lo: 0, Hi: 0}
}

// Equals returns true if u == v.
//
// Uint128 values can be compared directly with ==, but use of the Equals method
// is preferred for consistency.
func (u Uint128) Equals(v Uint128) bool {
  return u == v
}

// Equals64 returns true if u == v. - not used
func (u Uint128) Equals64(v uint64) bool {
  return u.Lo == v && u.Hi == 0
}

// Cmp compares u and v and returns: - not used
//
//   -1 if u <  v
//    0 if u == v
//   +1 if u >  v
//
func (u Uint128) Cmp(v Uint128) int {
  if u == v {
    return 0
  } else if u.Hi < v.Hi || (u.Hi == v.Hi && u.Lo < v.Lo) {
    return -1
  } else {
    return 1
  }
}

// Cmp64 compares u and v and returns: - not used
//
//   -1 if u <  v
//    0 if u == v
//   +1 if u >  v
//
func (u Uint128) Cmp64(v uint64) int {
  if u.Hi == 0 && u.Lo == v {
    return 0
  } else if u.Hi == 0 && u.Lo < v {
    return -1
  } else {
    return 1
  }
}

// And returns u&v.
func (u Uint128) And(v Uint128) Uint128 {
  return Uint128{Lo: u.Lo & v.Lo, Hi: u.Hi & v.Hi}
}

// And64 returns u&v. - not used
func (u Uint128) And64(v uint64) Uint128 {
  return Uint128{Lo: u.Lo & v, Hi: 0}
}

// Or returns u|v.
func (u Uint128) Or(v Uint128) Uint128 {
  return Uint128{Lo: u.Lo | v.Lo, Hi: u.Hi | v.Hi}
}

// Or64 returns u|v. - not used
func (u Uint128) Or64(v uint64) Uint128 {
  return Uint128{Lo: u.Lo | v, Hi: u.Hi}
}

// Xor returns u^v.
func (u Uint128) Xor(v Uint128) Uint128 {
  return Uint128{Lo: u.Lo ^ v.Lo, Hi: u.Hi ^ v.Hi}
}

// Xor64 returns u^v. - not used
func (u Uint128) Xor64(v uint64) Uint128 {
  return Uint128{Lo: u.Lo ^ v, Hi: u.Hi ^ 0}
}

// Add returns u+v. - not used
func (u Uint128) Add(v Uint128) Uint128 {
  lo, carry := bits.Add64(u.Lo, v.Lo, 0)
  hi, carry := bits.Add64(u.Hi, v.Hi, carry)
  if carry != 0 {
    panic("overflow")
  }
  return Uint128{Lo: lo, Hi: hi}
}

// AddWrap returns u+v with wraparound semantics; for example,
// Max.AddWrap(From64(1)) == Zero. - not used
func (u Uint128) AddWrap(v Uint128) Uint128 {
  lo, carry := bits.Add64(u.Lo, v.Lo, 0)
  hi, _ := bits.Add64(u.Hi, v.Hi, carry)
  return Uint128{Lo: lo, Hi: hi}
}

// Add64 returns u+v. - not used
func (u Uint128) Add64(v uint64) Uint128 {
  lo, carry := bits.Add64(u.Lo, v, 0)
  hi, carry := bits.Add64(u.Hi, 0, carry)
  if carry != 0 {
    panic("overflow")
  }
  return Uint128{Lo: lo, Hi: hi}
}

// AddWrap64 returns u+v with wraparound semantics; for example,
// Max.AddWrap64(1) == Zero. - not used
func (u Uint128) AddWrap64(v uint64) Uint128 {
  lo, carry := bits.Add64(u.Lo, v, 0)
  hi := u.Hi + carry
  return Uint128{Lo: lo, Hi: hi}
}

// Sub returns u-v.
func (u Uint128) Sub(v Uint128) Uint128 {
  lo, borrow := bits.Sub64(u.Lo, v.Lo, 0)
  hi, borrow := bits.Sub64(u.Hi, v.Hi, borrow)
  if borrow != 0 {
    panic("underflow")
  }
  return Uint128{Lo: lo, Hi: hi}
}

// SubWrap returns u-v with wraparound semantics; for example,
// Zero.SubWrap(From64(1)) == Max. - not used
func (u Uint128) SubWrap(v Uint128) Uint128 {
  lo, borrow := bits.Sub64(u.Lo, v.Lo, 0)
  hi, _ := bits.Sub64(u.Hi, v.Hi, borrow)
  return Uint128{Lo: lo, Hi: hi}
}

// Sub64 returns u-v. - not used
func (u Uint128) Sub64(v uint64) Uint128 {
  lo, borrow := bits.Sub64(u.Lo, v, 0)
  hi, borrow := bits.Sub64(u.Hi, 0, borrow)
  if borrow != 0 {
    panic("underflow")
  }
  return Uint128{Lo: lo, Hi: hi}
}

// SubWrap64 returns u-v with wraparound semantics; for example,
// Zero.SubWrap64(1) == Max. - not used
func (u Uint128) SubWrap64(v uint64) Uint128 {
  lo, borrow := bits.Sub64(u.Lo, v, 0)
  hi := u.Hi - borrow
  return Uint128{Lo: lo, Hi: hi}
}

// Lsh returns u<<n.
func (u Uint128) Lsh(n uint) (s Uint128) {
  if n > 64 {
    s.Lo = 0
    s.Hi = u.Lo << (n - 64)
  } else {
    s.Lo = u.Lo << n
    s.Hi = u.Hi<<n | u.Lo>>(64-n)
  }
  return
}

// Rsh returns u>>n. - not used
func (u Uint128) Rsh(n uint) (s Uint128) {
  if n > 64 {
    s.Lo = u.Hi >> (n - 64)
    s.Hi = 0
  } else {
    s.Lo = u.Lo>>n | u.Hi<<(64-n)
    s.Hi = u.Hi >> n
  }
  return
}

// LeadingZeros returns the number of leading zero bits in u; the result is 128
// for u == 0. - not used
func (u Uint128) LeadingZeros() int {
  if u.Hi > 0 {
    return bits.LeadingZeros64(u.Hi)
  }
  return 64 + bits.LeadingZeros64(u.Lo)
}

// TrailingZeros returns the number of trailing zero bits in u; the result is
// 128 for u == 0. - not used
func (u Uint128) TrailingZeros() int {
  if u.Lo > 0 {
    return bits.TrailingZeros64(u.Lo)
  }
  return 64 + bits.TrailingZeros64(u.Hi)
}

// OnesCount returns the number of one bits ("population count") in u.
func (u Uint128) OnesCount() int {
  return bits.OnesCount64(u.Hi) + bits.OnesCount64(u.Lo)
}

// Len returns the minimum number of bits required to represent u; the result is
// 0 for u == 0.
func (u Uint128) Len() int {
  return 128 - u.LeadingZeros()
}

// PutBytes stores u in b in little-endian order. It panics if len(b) < 16.
// - not used
func (u Uint128) PutBytes(b []byte) {
  binary.LittleEndian.PutUint64(b[:8], u.Lo)
  binary.LittleEndian.PutUint64(b[8:], u.Hi)
}

// Scan implements fmt.Scanner. - not used
func (u *Uint128) Scan(s fmt.ScanState, ch rune) error {
  i := new(big.Int)
  if err := i.Scan(s, ch); err != nil {
    return err
  } else if i.Sign() < 0 {
    return errors.New("value cannot be negative")
  } else if i.BitLen() > 128 {
    return errors.New("value overflows Uint128")
  }
  u.Lo = i.Uint64()
  u.Hi = i.Rsh(i, 64).Uint64()
  return nil
}

// New returns the Uint128 value (lo,hi).
func New(lo, hi uint64) Uint128 {
  return Uint128{Lo: lo, Hi: hi}
}

// From64 converts v to a Uint128 value.
func From64(v uint64) Uint128 {
  return New(v, 0)
}

// FromBytes converts b to a Uint128 value. - not used
func FromBytes(b []byte) Uint128 {
  return New(
    binary.LittleEndian.Uint64(b[:8]),
    binary.LittleEndian.Uint64(b[8:]),
  )
}

// FromString parses s as a Uint128 value. - not used
func FromString(s string) (u Uint128, err error) {
  _, err = fmt.Sscan(s, &u)
  return
}


// wpl add octal and hex output
// Convert input to octal string
// only 81 bits are taken into account
var MASK = From64(511) // 0o777 = 0x1ff = 511
func (u Uint128) ToOctal() string {

    // local variables
    var str[] string
    var result2[] string

    temp := u
    // extract 9 bits at a time
    for i := 0; i < 9; i++ {
        str = append(str, fmt.Sprintf("%03o", temp.And(MASK).Lo))
        temp = temp.Rsh(9)
    }

    // reverse 'str'
    for i := 8; i >= 0; i-- {
        result2 = append(result2, str[i])
    }
    return strings.Join(result2, " ")
}

// Convert u to hex string
// only 84 bits are taken into account
func (u Uint128) ToHex() string {
    return fmt.Sprintf("%05x %016x", u.Hi, u.Lo)
}

// We need the complement of an Uint128 input value
// convert 0 -> 1 and 1 -> 0
func (u Uint128) Not() Uint128 {
    return Uint128{Lo: u.Lo ^ math.MaxUint64, Hi: u.Hi ^ math.MaxUint64}
}

// sort comparison function
// Less compares u and v and returns:
//
//    true if u <  v
//    false otherwise
//
func (u Uint128) Less(v Uint128) bool {
    if u == v {
        return false
    } else if u.Hi < v.Hi || (u.Hi == v.Hi && u.Lo < v.Lo) {
        return true
    } else {
        return false
    }
}
