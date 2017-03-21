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

import "fmt"

type ObjectIterator struct {
	s        Slice
	position ValueLength
	size     ValueLength
	current  Slice
}

// NewObjectIterator initializes an iterator at position 0 of the given object slice.
func NewObjectIterator(s Slice) (*ObjectIterator, error) {
	if !s.IsObject() {
		return nil, InvalidTypeError{"Expected Object slice"}
	}
	size, err := s.Length()
	if err != nil {
		return nil, WithStack(err)
	}
	i := &ObjectIterator{
		s:        s,
		position: 0,
		size:     size,
	}
	if size > 0 {
		if h := s.head(); h == 0x14 {
			i.current, err = s.KeyAt(0, false)
		} else {
			// _current = slice.begin() + slice.findDataOffset(h);
			// TODO
			return nil, fmt.Errorf("TODO")
		}
	}
	return i, nil
}

// IsValid returns true if the given position of the iterator is valid.
func (i *ObjectIterator) IsValid() bool {
	return i.position < i.size
}

// IsFirst returns true if the current position is 0.
func (i *ObjectIterator) IsFirst() bool {
	return i.position == 0
}

// Key returns the key of the current position of the iterator
func (i *ObjectIterator) Key(translate bool) (Slice, error) {
	if i.position >= i.size {
		return nil, WithStack(IndexOutOfBoundsError{})
	}
	if current := i.current; current != nil {
		if translate {
			key, err := current.makeKey()
			return key, WithStack(err)
		}
		return current, nil
	}
	key, err := i.s.getNthKey(i.position, translate)
	return key, WithStack(err)
}

// MustKey returns the key of the current position of the iterator.
// Panics in case of an error.
func (i *ObjectIterator) MustKey(translate bool) Slice {
	if result, err := i.Key(translate); err != nil {
		panic(err)
	} else {
		return result
	}
}

// Value returns the value of the current position of the iterator
func (i *ObjectIterator) Value() (Slice, error) {
	if i.position >= i.size {
		return nil, WithStack(IndexOutOfBoundsError{})
	}
	if current := i.current; current != nil {
		value, err := current.Next()
		return value, WithStack(err)
	}
	value, err := i.s.getNthValue(i.position)
	return value, WithStack(err)
}

// MustValue returns the value of the current position of the iterator.
// Panics in case of an error.
func (i *ObjectIterator) MustValue() Slice {
	if result, err := i.Value(); err != nil {
		panic(err)
	} else {
		return result
	}
}

// Next moves to the next position.
func (i *ObjectIterator) Next() error {
	i.position++
	if i.position < i.size && i.current != nil {
		var err error
		// skip over key
		i.current, err = i.current.Next()
		if err != nil {
			return WithStack(err)
		}
		// skip over value
		i.current, err = i.current.Next()
		if err != nil {
			return WithStack(err)
		}
	} else {
		i.current = nil
	}
	return nil
}