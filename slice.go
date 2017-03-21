//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//
// Author Ewout Prangsma
//

package velocypack

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math"
)

// Slice provides read only access to a VPack value
type Slice []byte

// SliceFromBytes creates a Slice by casting the given byte slice to a Slice.
func SliceFromBytes(v []byte) Slice {
	return Slice(v)
}

// SliceFromHex creates a Slice by decoding the given hex string into a Slice.
// If decoding fails, nil is returned.
func SliceFromHex(v string) Slice {
	if bytes, err := hex.DecodeString(v); err != nil {
		return nil
	} else {
		return Slice(bytes)
	}
}

// String returns a HEX representation of the slice.
func (s Slice) String() string {
	return hex.EncodeToString(s)
}

// JSONString converts the contents of the slice to JSON.
func (s Slice) JSONString() (string, error) {
	buf := &bytes.Buffer{}
	d := NewDumper(buf)
	if err := d.Append(s); err != nil {
		return "", WithStack(err)
	}
	return buf.String(), nil
}

// MustJSONString converts the contents of the slice to JSON.
// Panics in case of an error.
func (s Slice) MustJSONString() string {
	if result, err := s.JSONString(); err != nil {
		panic(err)
	} else {
		return result
	}
}

// head returns the first element of the slice or 0 if the slice is empty.
func (s Slice) head() byte {
	if len(s) > 0 {
		return s[0]
	}
	return 0
}

// ByteSize returns the total byte size for the slice, including the head byte
func (s Slice) ByteSize() (ValueLength, error) {
	h := s.head()
	// check if the type has a fixed length first
	l := fixedTypeLengths[h]
	if l != 0 {
		// return fixed length
		return ValueLength(l), nil
	}

	// types with dynamic lengths need special treatment:
	switch s.Type() {
	case Array, Object:
		if h == 0x13 || h == 0x14 {
			// compact Array or Object
			return readVariableValueLength(s[1:], false), nil
		}

		if h == 0x01 || h == 0x0a {
			// we cannot get here, because the FixedTypeLengths lookup
			// above will have kicked in already. however, the compiler
			// claims we'll be reading across the bounds of the input
			// here...
			return 1, nil
		}

		VELOCYPACK_ASSERT(h > 0x00 && h <= 0x0e)
		return ValueLength(readIntegerNonEmpty(s[1:], widthMap[h])), nil

	case External:
		return 1 + charPtrLength, nil

	case UTCDate:
		return 1 + int64Length, nil

	case Int:
		return ValueLength(1 + (h - 0x1f)), nil

	case String:
		VELOCYPACK_ASSERT(h == 0xbf)
		if h < 0xbf {
			// we cannot get here, because the FixedTypeLengths lookup
			// above will have kicked in already. however, the compiler
			// claims we'll be reading across the bounds of the input
			// here...
			return ValueLength(h) - 0x40, nil
		}
		// long UTF-8 String
		return ValueLength(1 + 8 + readIntegerFixed(s[1:], 8)), nil

	case Binary:
		VELOCYPACK_ASSERT(h >= 0xc0 && h <= 0xc7)
		return ValueLength(1 + ValueLength(h) - 0xbf + ValueLength(readIntegerNonEmpty(s[1:], uint(h)-0xbf))), nil

	case BCD:
		if h <= 0xcf {
			// positive BCD
			VELOCYPACK_ASSERT(h >= 0xc8 && h < 0xcf)
			return ValueLength(1 + ValueLength(h) - 0xc7 + ValueLength(readIntegerNonEmpty(s[1:], uint(h)-0xc7))), nil
		}

		// negative BCD
		VELOCYPACK_ASSERT(h >= 0xd0 && h < 0xd7)
		return ValueLength(1 + ValueLength(h) - 0xcf + ValueLength(readIntegerNonEmpty(s[1:], uint(h)-0xcf))), nil

	case Custom:
		VELOCYPACK_ASSERT(h >= 0xf4)
		switch h {
		case 0xf4, 0xf5, 0xf6:
			return ValueLength(2 + readIntegerFixed(s[1:], 1)), nil
		case 0xf7, 0xf8, 0xf9:
			return ValueLength(3 + readIntegerFixed(s[1:], 2)), nil
		case 0xfa, 0xfb, 0xfc:
			return ValueLength(5 + readIntegerFixed(s[1:], 4)), nil
		case 0xfd, 0xfe, 0xff:
			return ValueLength(9 + readIntegerFixed(s[1:], 8)), nil
		}
	}

	return 0, InternalError{}
}

// MustByteSize returns the total byte size for the slice, including the head byte.
// Panics in case of an error.
func (s Slice) MustByteSize() ValueLength {
	if v, err := s.ByteSize(); err != nil {
		panic(err)
	} else {
		return v
	}
}

// Next returns the Slice that directly follows the given slice.
// Same as s[s.ByteSize:]
func (s Slice) Next() (Slice, error) {
	size, err := s.ByteSize()
	if err != nil {
		return nil, WithStack(err)
	}
	return Slice(s[size:]), nil
}

// MustNext returns the Slice that directly follows the given slice.
// Same as s[s.ByteSize:]
// Panics in case of an error.
func (s Slice) MustNext() Slice {
	if result, err := s.Next(); err != nil {
		panic(err)
	} else {
		return result
	}
}

// GetBool returns a boolean value from the slice.
// Returns an error if slice is not of type Bool.
func (s Slice) GetBool() (bool, error) {
	if err := s.AssertType(Bool); err != nil {
		return false, WithStack(err)
	}
	return s.IsTrue(), nil
}

// MustGetBool returns a boolean value from the slice.
// Panics if slice is not of type Bool.
func (s Slice) MustGetBool() bool {
	if v, err := s.GetBool(); err != nil {
		panic(err)
	} else {
		return v
	}
}

// GetDouble returns a Double value from the slice.
// Returns an error if slice is not of type Double.
func (s Slice) GetDouble() (float64, error) {
	if err := s.AssertType(Double); err != nil {
		return 0.0, WithStack(err)
	}
	bits := binary.LittleEndian.Uint64(s[1:])
	return math.Float64frombits(bits), nil
}

// MustGetDouble returns a Double value from the slice.
// Panics if slice is not of type Double.
func (s Slice) MustGetDouble() float64 {
	if v, err := s.GetDouble(); err != nil {
		panic(err)
	} else {
		return v
	}
}

// GetInt returns a Int value from the slice.
// Returns an error if slice is not of type Int.
func (s Slice) GetInt() (int64, error) {
	h := s.head()

	if h >= 0x20 && h <= 0x27 {
		// Int  T
		v := readIntegerNonEmpty(s[1:], uint(h)-0x1f)
		if h == 0x27 {
			return toInt64(v), nil
		} else {
			vv := int64(v)
			shift := int64(1) << ((h-0x1f)*8 - 1)
			if vv < shift {
				return vv, nil
			} else {
				return vv - (shift << 1), nil
			}
		}
	}

	if h >= 0x28 && h <= 0x2f {
		// UInt
		v, err := s.GetUInt()
		if err != nil {
			return 0, WithStack(err)
		}
		if v > math.MaxInt64 {
			return 0, NumberOutOfRangeError{}
		}
		return int64(v), nil
	}

	if h >= 0x30 && h <= 0x3f {
		// SmallInt
		return s.GetSmallInt()
	}

	return 0, InvalidTypeError{"Expecting type Int"}
}

// MustGetInt returns a Int value from the slice.
// Panics if slice is not of type Int.
func (s Slice) MustGetInt() int64 {
	if v, err := s.GetInt(); err != nil {
		panic(err)
	} else {
		return v
	}
}

// GetUInt returns a UInt value from the slice.
// Returns an error if slice is not of type UInt.
func (s Slice) GetUInt() (uint64, error) {
	h := s.head()

	if h == 0x28 {
		// single byte integer
		return uint64(s[1]), nil
	}

	if h >= 0x29 && h <= 0x2f {
		// UInt
		return readIntegerNonEmpty(s[1:], uint(h)-0x27), nil
	}

	if h >= 0x20 && h <= 0x27 {
		// Int
		v, err := s.GetInt()
		if err != nil {
			return 0, WithStack(err)
		}
		if v < 0 {
			return 0, NumberOutOfRangeError{}
		}
		return uint64(v), nil
	}

	if h >= 0x30 && h <= 0x39 {
		// Smallint >= 0
		return uint64(h - 0x30), nil
	}

	if h >= 0x3a && h <= 0x3f {
		// Smallint < 0
		return 0, NumberOutOfRangeError{}
	}

	return 0, InvalidTypeError{"Expecting type UInt"}
}

// MustGetUInt returns a UInt value from the slice.
// Panics if slice is not of type UInt.
func (s Slice) MustGetUInt() uint64 {
	if v, err := s.GetUInt(); err != nil {
		panic(err)
	} else {
		return v
	}
}

// GetSmallInt returns a SmallInt value from the slice.
// Returns an error if slice is not of type SmallInt.
func (s Slice) GetSmallInt() (int64, error) {
	h := s.head()

	if h >= 0x30 && h <= 0x39 {
		// Smallint >= 0
		return int64(h - 0x30), nil
	}

	if h >= 0x3a && h <= 0x3f {
		// Smallint < 0
		return int64(h-0x3a) - 6, nil
	}

	if (h >= 0x20 && h <= 0x27) || (h >= 0x28 && h <= 0x2f) {
		// Int and UInt
		// we'll leave it to the compiler to detect the two ranges above are
		// adjacent
		return s.GetInt()
	}

	return 0, InvalidTypeError{"Expecting type SmallInt"}
}

// MustGetSmallInt returns a SmallInt value from the slice.
// Panics if slice is not of type SmallInt.
func (s Slice) MustGetSmallInt() int64 {
	if v, err := s.GetSmallInt(); err != nil {
		panic(err)
	} else {
		return v
	}
}

// GetString return the value for a String object
func (s Slice) GetString() (string, error) {
	h := s.head()
	if h >= 0x40 && h <= 0xbe {
		// short UTF-8 String
		length := h - 0x40
		result := string(s[1 : 1+length])
		return result, nil
	}

	if h == 0xbf {
		// long UTF-8 String
		length := readIntegerFixed(s[1:], 8)
		if err := checkOverflow(ValueLength(length)); err != nil {
			return "", WithStack(err)
		}
		result := string(s[1+8 : 1+8+length])
		return result, nil
	}

	return "", InvalidTypeError{"Expecting type String"}
}

// MustGetString return the value for a String object.
// Panics in case of an error.
func (s Slice) MustGetString() string {
	if result, err := s.GetString(); err != nil {
		panic(err)
	} else {
		return result
	}
}

// GetStringLength return the length for a String object
func (s Slice) GetStringLength() (ValueLength, error) {
	h := s.head()
	if h >= 0x40 && h <= 0xbe {
		// short UTF-8 String
		length := h - 0x40
		return ValueLength(length), nil
	}

	if h == 0xbf {
		// long UTF-8 String
		length := readIntegerFixed(s[1:], 8)
		if err := checkOverflow(ValueLength(length)); err != nil {
			return 0, WithStack(err)
		}
		return ValueLength(length), nil
	}

	return 0, InvalidTypeError{"Expecting type String"}
}

// MustGetStringLength return the length for a String object.
// Panics in case of an error.
func (s Slice) MustGetStringLength() ValueLength {
	if result, err := s.GetStringLength(); err != nil {
		panic(err)
	} else {
		return result
	}
}

// CompareString compares the string value in the slice with the given string.
// s == value -> 0
// s < value -> -1
// s > value -> 1
func (s Slice) CompareString(value string) (int, error) {
	k, err := s.GetString()
	if err != nil {
		return 0, WithStack(err)
	}
	return bytes.Compare([]byte(k), []byte(value)), nil
}

// MustCompareString compares the string value in the slice with the given string.
// s == value -> 0
// s < value -> -1
// s > value -> 1
// Panics in case of an error.
func (s Slice) MustCompareString(value string) int {
	if result, err := s.CompareString(value); err != nil {
		panic(err)
	} else {
		return result
	}
}

// IsEqualString compares the string value in the slice with the given string for equivalence.
func (s Slice) IsEqualString(value string) (bool, error) {
	k, err := s.GetString()
	if err != nil {
		return false, WithStack(err)
	}
	return k == value, nil
}

// MustIsEqualString compares the string value in the slice with the given string for equivalence.
// Panics in case of an error.
func (s Slice) MustIsEqualString(value string) bool {
	if result, err := s.IsEqualString(value); err != nil {
		panic(err)
	} else {
		return result
	}
}

// GetBinary return the value for a Binary object
func (s Slice) GetBinary() ([]byte, error) {
	if !s.IsBinary() {
		return nil, InvalidTypeError{"Expecting type Binary"}
	}

	h := s.head()
	VELOCYPACK_ASSERT(h >= 0xc0 && h <= 0xc7)

	lengthSize := uint(h - 0xbf)
	length := readIntegerNonEmpty(s[1:], lengthSize)
	checkOverflow(ValueLength(length))
	return s[1+lengthSize : 1+uint64(lengthSize)+length], nil
}

// MustGetBinary return the value for a Binary object.
// Panics in case of an error.
func (s Slice) MustGetBinary() []byte {
	if result, err := s.GetBinary(); err != nil {
		panic(err)
	} else {
		return result
	}
}

// GetBinaryLength return the length for a Binary object
func (s Slice) GetBinaryLength() (ValueLength, error) {
	if !s.IsBinary() {
		return 0, InvalidTypeError{"Expecting type Binary"}
	}

	h := s.head()
	VELOCYPACK_ASSERT(h >= 0xc0 && h <= 0xc7)

	lengthSize := uint(h - 0xbf)
	length := readIntegerNonEmpty(s[1:], lengthSize)
	return ValueLength(length), nil
}

// MustGetBinaryLength return the length for a Binary object.
// Panics in case of an error.
func (s Slice) MustGetBinaryLength() ValueLength {
	if result, err := s.GetBinaryLength(); err != nil {
		panic(err)
	} else {
		return result
	}
}

// Length return the number of members for an Array or Object object
func (s Slice) Length() (ValueLength, error) {
	if !s.IsArray() && !s.IsObject() {
		return 0, InvalidTypeError{"Expecting type Array or Object"}
	}

	h := s.head()
	if h == 0x01 || h == 0x0a {
		// special case: empty!
		return 0, nil
	}

	if h == 0x13 || h == 0x14 {
		// compact Array or Object
		end := readVariableValueLength(s[1:], false)
		return readVariableValueLength(s[end-1:], true), nil
	}

	offsetSize := indexEntrySize(h)
	VELOCYPACK_ASSERT(offsetSize > 0)
	end := readIntegerNonEmpty(s[1:], offsetSize)

	// find number of items
	if h <= 0x05 { // No offset table or length, need to compute:
		firstSubOffset := s.findDataOffset(h)
		first := s[firstSubOffset:]
		s, err := first.ByteSize()
		if err != nil {
			return 0, WithStack(err)
		}
		if s == 0 {
			return 0, InternalError{}
		}
		return (ValueLength(end) - firstSubOffset) / s, nil
	} else if offsetSize < 8 {
		return ValueLength(readIntegerNonEmpty(s[offsetSize+1:], offsetSize)), nil
	}

	return ValueLength(readIntegerNonEmpty(s[end-uint64(offsetSize):], offsetSize)), nil
}

// MustLength return the number of members for an Array or Object object.
// Panics in case of error.
func (s Slice) MustLength() ValueLength {
	if result, err := s.Length(); err != nil {
		panic(err)
	} else {
		return result
	}
}

// At extracts the array value at the specified index.
func (s Slice) At(index ValueLength) (Slice, error) {
	if !s.IsArray() {
		return nil, InvalidTypeError{"Expecting type Array"}
	}

	if result, err := s.getNth(index); err != nil {
		return nil, WithStack(err)
	} else {
		return result, nil
	}
}

// MustAt extracts the array value at the specified index.
// Panics in case of an error.
func (s Slice) MustAt(index ValueLength) Slice {
	if result, err := s.At(index); err != nil {
		panic(err)
	} else {
		return result
	}
}

// KeyAt extracts a key from an Object at the specified index.
func (s Slice) KeyAt(index ValueLength, translate ...bool) (Slice, error) {
	if !s.IsObject() {
		return nil, InvalidTypeError{"Expecting type Object"}
	}

	return s.getNthKey(index, optionalBool(translate, true))
}

// MustKeyAt extracts a key from an Object at the specified index.
// Panics in case of an error.
func (s Slice) MustKeyAt(index ValueLength, translate ...bool) Slice {
	if result, err := s.KeyAt(index, translate...); err != nil {
		panic(err)
	} else {
		return result
	}
}

// ValueAt extracts a value from an Object at the specified index
func (s Slice) ValueAt(index ValueLength) (Slice, error) {
	if !s.IsObject() {
		return nil, InvalidTypeError{"Expecting type Object"}
	}

	key, err := s.getNthKey(index, false)
	if err != nil {
		return nil, WithStack(err)
	}
	byteSize, err := key.ByteSize()
	if err != nil {
		return nil, WithStack(err)
	}
	return Slice(key[byteSize:]), nil
}

// MustValueAt extracts a value from an Object at the specified index.
// Panics in case of an error.
func (s Slice) MustValueAt(index ValueLength) Slice {
	if result, err := s.ValueAt(index); err != nil {
		panic(err)
	} else {
		return result
	}
}

func indexEntrySize(head byte) uint {
	VELOCYPACK_ASSERT(head > 0x00 && head <= 0x12)
	return widthMap[head]
}

// Get looks for the specified attribute inside an Object
// returns a Slice(ValueType::None) if not found
func (s Slice) Get(attribute string) (Slice, error) {
	if !s.IsObject() {
		return nil, InvalidTypeError{"Expecting Object"}
	}

	h := s.head()
	if h == 0x0a {
		// special case, empty object
		return nil, nil
	}

	if h == 0x14 {
		// compact Object
		value, err := s.getFromCompactObject(attribute)
		return value, WithStack(err)
	}

	offsetSize := indexEntrySize(h)
	VELOCYPACK_ASSERT(offsetSize > 0)
	end := ValueLength(readIntegerNonEmpty(s[1:], offsetSize))

	// read number of items
	var n ValueLength
	var ieBase ValueLength
	if offsetSize < 8 {
		n = ValueLength(readIntegerNonEmpty(s[1+offsetSize:], offsetSize))
		ieBase = end - n*ValueLength(offsetSize)
	} else {
		n = ValueLength(readIntegerNonEmpty(s[end-ValueLength(offsetSize):], offsetSize))
		ieBase = end - n*ValueLength(offsetSize) - ValueLength(offsetSize)
	}

	if n == 1 {
		// Just one attribute, there is no index table!
		key := Slice(s[s.findDataOffset(h):])

		if key.IsString() {
			if eq, err := key.IsEqualString(attribute); err != nil {
				return nil, WithStack(err)
			} else if eq {
				value, err := key.Next()
				return value, WithStack(err)
			}
			// fall through to returning None Slice below
		} else if key.IsSmallInt() || key.IsUInt() {
			// translate key
			if AttributeTranslator == nil {
				return nil, WithStack(NeedAttributeTranslatorError{})
			}
			if eq, err := key.translateUnchecked().IsEqualString(attribute); err != nil {
				return nil, WithStack(err)
			} else if eq {
				value, err := key.Next()
				return value, WithStack(err)
			}
		}

		// no match or invalid key type
		return nil, nil
	}

	// only use binary search for attributes if we have at least this many entries
	// otherwise we'll always use the linear search
	const SortedSearchEntriesThreshold = ValueLength(4)

	// bool const isSorted = (h >= 0x0b && h <= 0x0e);
	if n >= SortedSearchEntriesThreshold && (h >= 0x0b && h <= 0x0e) {
		// This means, we have to handle the special case n == 1 only
		// in the linear search!
		switch offsetSize {
		case 1:
			result, err := s.searchObjectKeyBinary(attribute, ieBase, n, 1)
			return result, WithStack(err)
		case 2:
			result, err := s.searchObjectKeyBinary(attribute, ieBase, n, 2)
			return result, WithStack(err)
		case 4:
			result, err := s.searchObjectKeyBinary(attribute, ieBase, n, 4)
			return result, WithStack(err)
		case 8:
			result, err := s.searchObjectKeyBinary(attribute, ieBase, n, 8)
			return result, WithStack(err)
		}
	}

	result, err := s.searchObjectKeyLinear(attribute, ieBase, ValueLength(offsetSize), n)
	return result, WithStack(err)
}

// MustGet looks for the specified attribute inside an Object
// returns a Slice(ValueType::None) if not found
// Panics in case of an error.
func (s Slice) MustGet(attribute string) Slice {
	if result, err := s.Get(attribute); err != nil {
		panic(err)
	} else {
		return result
	}
}

func (s Slice) getFromCompactObject(attribute string) (Slice, error) {
	it, err := NewObjectIterator(s)
	if err != nil {
		return nil, WithStack(err)
	}
	for it.IsValid() {
		key, err := it.Key(false)
		if err != nil {
			return nil, WithStack(err)
		}
		k, err := key.makeKey()
		if err != nil {
			return nil, WithStack(err)
		}
		if eq, err := k.IsEqualString(attribute); err != nil {
			return nil, WithStack(err)
		} else if eq {
			value, err := key.Next()
			return value, WithStack(err)
		}

		if err := it.Next(); err != nil {
			return nil, WithStack(err)
		}
	}
	// not found
	return nil, nil
}

func (s Slice) findDataOffset(head byte) ValueLength {
	// Must be called for a nonempty array or object at start():
	VELOCYPACK_ASSERT(head <= 0x12)
	fsm := firstSubMap[head]
	if fsm <= 2 && s[2] != 0 {
		return 2
	}
	if fsm <= 3 && s[3] != 0 {
		return 3
	}
	if fsm <= 5 && s[5] != 0 {
		return 5
	}
	return 9
}

// get the offset for the nth member from an Array or Object type
func (s Slice) getNthOffset(index ValueLength) (ValueLength, error) {
	VELOCYPACK_ASSERT(s.IsArray() || s.IsObject())

	h := s.head()

	if h == 0x13 || h == 0x14 {
		// compact Array or Object
		l, err := s.getNthOffsetFromCompact(index)
		if err != nil {
			return 0, WithStack(err)
		}
		return l, nil
	}

	if h == 0x01 || h == 0x0a {
		// special case: empty Array or empty Object
		return 0, IndexOutOfBoundsError{}
	}

	offsetSize := indexEntrySize(h)
	end := ValueLength(readIntegerNonEmpty(s[1:], offsetSize))

	dataOffset := ValueLength(0)

	// find the number of items
	var n ValueLength
	if h <= 0x05 { // No offset table or length, need to compute:
		dataOffset = s.findDataOffset(h)
		first := Slice(s[dataOffset:])
		s, err := first.ByteSize()
		if err != nil {
			return 0, WithStack(err)
		}
		if s == 0 {
			return 0, InternalError{}
		}
		n = (end - dataOffset) / s
	} else if offsetSize < 8 {
		n = ValueLength(readIntegerNonEmpty(s[1+offsetSize:], offsetSize))
	} else {
		n = ValueLength(readIntegerNonEmpty(s[end-ValueLength(offsetSize):], offsetSize))
	}

	if index >= n {
		return 0, IndexOutOfBoundsError{}
	}

	// empty array case was already covered
	VELOCYPACK_ASSERT(n > 0)

	if h <= 0x05 || n == 1 {
		// no index table, but all array items have the same length
		// now fetch first item and determine its length
		if dataOffset == 0 {
			dataOffset = s.findDataOffset(h)
		}
		sliceAtDataOffset := Slice(s[dataOffset:])
		sliceAtDataOffsetByteSize, err := sliceAtDataOffset.ByteSize()
		if err != nil {
			return 0, WithStack(err)
		}
		return dataOffset + index*sliceAtDataOffsetByteSize, nil
	}

	offsetSize8Or0 := ValueLength(0)
	if offsetSize == 8 {
		offsetSize8Or0 = 8
	}
	ieBase := end - n*ValueLength(offsetSize) + index*ValueLength(offsetSize) - (offsetSize8Or0)
	return ValueLength(readIntegerNonEmpty(s[ieBase:], offsetSize)), nil
}

// get the offset for the nth member from a compact Array or Object type
func (s Slice) getNthOffsetFromCompact(index ValueLength) (ValueLength, error) {
	end := ValueLength(readVariableValueLength(s[1:], false))
	n := ValueLength(readVariableValueLength(s[end-1:], true))
	if index >= n {
		return 0, IndexOutOfBoundsError{}
	}

	h := s.head()
	offset := ValueLength(1 + getVariableValueLength(end))
	current := ValueLength(0)
	for current != index {
		sliceAtOffset := Slice(s[offset:])
		sliceAtOffsetByteSize, err := sliceAtOffset.ByteSize()
		if err != nil {
			return 0, WithStack(err)
		}
		offset += sliceAtOffsetByteSize
		if h == 0x14 {
			sliceAtOffset := Slice(s[offset:])
			sliceAtOffsetByteSize, err := sliceAtOffset.ByteSize()
			if err != nil {
				return 0, WithStack(err)
			}
			offset += sliceAtOffsetByteSize
		}
		current++
	}
	return offset, nil
}

// extract the nth member from an Array
func (s Slice) getNth(index ValueLength) (Slice, error) {
	VELOCYPACK_ASSERT(s.IsArray())

	offset, err := s.getNthOffset(index)
	if err != nil {
		return nil, WithStack(err)
	}
	return Slice(s[offset:]), nil
}

// getNthKey extract the nth member from an Object
func (s Slice) getNthKey(index ValueLength, translate bool) (Slice, error) {
	VELOCYPACK_ASSERT(s.Type() == Object)

	offset, err := s.getNthOffset(index)
	if err != nil {
		return nil, WithStack(err)
	}
	result := Slice(s[offset:])
	if translate {
		result, err = result.makeKey()
		if err != nil {
			return nil, WithStack(err)
		}
	}
	return result, nil
}

// getNthValue extract the nth value from an Object
func (s Slice) getNthValue(index ValueLength) (Slice, error) {
	key, err := s.getNthKey(index, false)
	if err != nil {
		return nil, WithStack(err)
	}
	value, err := key.Next()
	return value, WithStack(err)
}

func (s Slice) makeKey() (Slice, error) {
	if s.IsString() {
		return s, nil
	}
	if s.IsSmallInt() || s.IsUInt() {
		if AttributeTranslator == nil {
			return nil, WithStack(NeedAttributeTranslatorError{})
		}
		return s.translateUnchecked(), nil
	}

	return nil, InvalidTypeError{"Cannot translate key of this type"}
}

// perform a linear search for the specified attribute inside an Object
func (s Slice) searchObjectKeyLinear(attribute string, ieBase, offsetSize, n ValueLength) (Slice, error) {
	useTranslator := AttributeTranslator != nil

	for index := ValueLength(0); index < n; index++ {
		offset := ValueLength(ieBase + index*offsetSize)
		key := Slice(s[readIntegerNonEmpty(s[offset:], uint(offsetSize)):])

		if key.IsString() {
			if eq, err := key.IsEqualString(attribute); err != nil {
				return nil, WithStack(err)
			} else if !eq {
				continue
			}
		} else if key.IsSmallInt() || key.IsUInt() {
			// translate key
			if !useTranslator {
				// no attribute translator
				return nil, WithStack(NeedAttributeTranslatorError{})
			}
			if eq, err := key.translateUnchecked().IsEqualString(attribute); err != nil {
				return nil, WithStack(err)
			} else if !eq {
				continue
			}
		} else {
			// invalid key type
			return nil, nil
		}

		// key is identical. now return value
		value, err := key.Next()
		return value, WithStack(err)
	}

	// nothing found
	return nil, nil
}

// perform a binary search for the specified attribute inside an Object
//template<ValueLength offsetSize>
func (s Slice) searchObjectKeyBinary(attribute string, ieBase ValueLength, n ValueLength, offsetSize ValueLength) (Slice, error) {
	useTranslator := AttributeTranslator != nil
	VELOCYPACK_ASSERT(n > 0)

	l := ValueLength(0)
	r := ValueLength(n - 1)
	index := ValueLength(r / 2)

	for {
		offset := ValueLength(ieBase + index*offsetSize)
		key := Slice(s[readIntegerFixed(s[offset:], uint(offsetSize)):])

		var res int
		var err error
		if key.IsString() {
			res, err = key.CompareString(attribute)
			if err != nil {
				return nil, WithStack(err)
			}
		} else if key.IsSmallInt() || key.IsUInt() {
			// translate key
			if !useTranslator {
				// no attribute translator
				return nil, NeedAttributeTranslatorError{}
			}
			res, err = key.translateUnchecked().CompareString(attribute)
			if err != nil {
				return nil, WithStack(err)
			}
		} else {
			// invalid key
			return nil, nil
		}

		if res == 0 {
			// found. now return a Slice pointing at the value
			keySize, err := key.ByteSize()
			if err != nil {
				return nil, WithStack(err)
			}
			return Slice(key[keySize:]), nil
		}

		if res > 0 {
			if index == 0 {
				return nil, nil
			}
			r = index - 1
		} else {
			l = index + 1
		}
		if r < l {
			return nil, nil
		}

		// determine new midpoint
		index = l + ((r - l) / 2)
	}
}

// translates an integer key into a string
func (s Slice) translate() (Slice, error) {
	if !s.IsSmallInt() && !s.IsUInt() {
		return nil, WithStack(InvalidTypeError{"Cannot translate key of this type"})
	}
	if AttributeTranslator == nil {
		return nil, WithStack(NeedAttributeTranslatorError{})
	}
	return s.translateUnchecked(), nil
}

// return the value for a UInt object, without checks!
// returns 0 for invalid values/types
func (s Slice) getUIntUnchecked() uint64 {
	h := s.head()
	if h >= 0x28 && h <= 0x2f {
		// UInt
		return readIntegerNonEmpty(s[1:], uint(h-0x27))
	}

	if h >= 0x30 && h <= 0x39 {
		// Smallint >= 0
		return uint64(h - 0x30)
	}
	return 0
}

// translates an integer key into a string, without checks
func (s Slice) translateUnchecked() Slice {
	id := s.getUIntUnchecked()
	key := AttributeTranslator.IDToString(id)
	if key == "" {
		return nil
	}
	return StringSlice(key)
}
