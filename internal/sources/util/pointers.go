// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package util provides common utility functions for source implementations.
package util

// Int32Ptr returns a pointer to the given int32 value.
func Int32Ptr(i int32) *int32 {
	return &i
}

// StringPtr returns a pointer to the given string value.
func StringPtr(s string) *string {
	return &s
}

// StringValue returns the value of a string pointer, or empty string if nil.
func StringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Float64Value returns the value of a float64 pointer, or 0 if nil.
func Float64Value(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

// Int32Value returns the value of an int32 pointer, or 0 if nil.
func Int32Value(i *int32) int32 {
	if i == nil {
		return 0
	}
	return *i
}

// Int64Ptr returns a pointer to the given int64 value.
func Int64Ptr(i int64) *int64 {
	return &i
}

// Int64Value returns the value of an int64 pointer, or 0 if nil.
func Int64Value(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}
