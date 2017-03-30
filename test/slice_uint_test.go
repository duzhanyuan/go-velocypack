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

package test

import (
	"testing"

	velocypack "github.com/arangodb/go-velocypack"
)

func TestSliceUInt1(t *testing.T) {
	slice := velocypack.Slice{0x28, 0x33}
	value := uint64(0x33)
	assertEqualFromReader(t, slice)

	ASSERT_EQ(velocypack.UInt, slice.Type(), t)
	ASSERT_TRUE(slice.IsUInt(), t)
	ASSERT_EQ(velocypack.ValueLength(2), slice.MustByteSize(), t)

	ASSERT_EQ(value, slice.MustGetUInt(), t)
}

func TestSliceUInt2(t *testing.T) {
	slice := velocypack.Slice{0x29, 0x23, 0x42}
	assertEqualFromReader(t, slice)

	ASSERT_EQ(velocypack.UInt, slice.Type(), t)
	ASSERT_TRUE(slice.IsUInt(), t)
	ASSERT_EQ(velocypack.ValueLength(3), slice.MustByteSize(), t)

	ASSERT_EQ(uint64(0x4223), slice.MustGetUInt(), t)
}

func TestSliceUInt3(t *testing.T) {
	slice := velocypack.Slice{0x2a, 0x23, 0x42, 0x66}
	assertEqualFromReader(t, slice)

	ASSERT_EQ(velocypack.UInt, slice.Type(), t)
	ASSERT_TRUE(slice.IsUInt(), t)
	ASSERT_EQ(velocypack.ValueLength(4), slice.MustByteSize(), t)

	ASSERT_EQ(uint64(0x664223), slice.MustGetUInt(), t)
}

func TestSliceUInt4(t *testing.T) {
	slice := velocypack.Slice{0x2b, 0x23, 0x42, 0x66, 0x7c}
	assertEqualFromReader(t, slice)

	ASSERT_EQ(velocypack.UInt, slice.Type(), t)
	ASSERT_TRUE(slice.IsUInt(), t)
	ASSERT_EQ(velocypack.ValueLength(5), slice.MustByteSize(), t)

	ASSERT_EQ(uint64(0x7c664223), slice.MustGetUInt(), t)
}

func TestSliceUInt5(t *testing.T) {
	slice := velocypack.Slice{0x2c, 0x23, 0x42, 0x66, 0xac, 0x6f}
	assertEqualFromReader(t, slice)

	ASSERT_EQ(velocypack.UInt, slice.Type(), t)
	ASSERT_TRUE(slice.IsUInt(), t)
	ASSERT_EQ(velocypack.ValueLength(6), slice.MustByteSize(), t)

	ASSERT_EQ(uint64(0x6fac664223), slice.MustGetUInt(), t)
}

func TestSliceUInt6(t *testing.T) {
	slice := velocypack.Slice{0x2d, 0x23, 0x42, 0x66, 0xac, 0xff, 0x3f}
	assertEqualFromReader(t, slice)

	ASSERT_EQ(velocypack.UInt, slice.Type(), t)
	ASSERT_TRUE(slice.IsUInt(), t)
	ASSERT_EQ(velocypack.ValueLength(7), slice.MustByteSize(), t)

	ASSERT_EQ(uint64(0x3fffac664223), slice.MustGetUInt(), t)
}

func TestSliceUInt7(t *testing.T) {
	slice := velocypack.Slice{0x2e, 0x23, 0x42, 0x66, 0xac, 0xff, 0x3f, 0x5a}
	assertEqualFromReader(t, slice)

	ASSERT_EQ(velocypack.UInt, slice.Type(), t)
	ASSERT_TRUE(slice.IsUInt(), t)
	ASSERT_EQ(velocypack.ValueLength(8), slice.MustByteSize(), t)

	ASSERT_EQ(uint64(0x5a3fffac664223), slice.MustGetUInt(), t)
}

func TestSliceUInt8(t *testing.T) {
	slice := velocypack.Slice{0x2f, 0x23, 0x42, 0x66, 0xac, 0xff, 0x3f, 0xfa, 0x6f}
	assertEqualFromReader(t, slice)

	ASSERT_EQ(velocypack.UInt, slice.Type(), t)
	ASSERT_TRUE(slice.IsUInt(), t)
	ASSERT_EQ(velocypack.ValueLength(9), slice.MustByteSize(), t)

	ASSERT_EQ(uint64(0x6ffa3fffac664223), slice.MustGetUInt(), t)
}

func TestSliceUIntMax(t *testing.T) {
	t.Skip("TODO")
	/*	  Builder b;
	  b.add(Value(INT64_MAX));

	  Slice slice(b.slice());

		ASSERT_EQ(velocypack.Int, slice.Type(), t)
		ASSERT_TRUE(slice.IsInt(), t)
		ASSERT_EQ(velocypack.ValueLength(9), slice.MustByteSize(), t)

		ASSERT_EQ(int64(math.MaxInt64), slice.MustGetInt(), t)
	*/
}
