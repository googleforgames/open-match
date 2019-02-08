/*
package apisrv provides an implementation of the gRPC server defined in ../../../api/protobuf-spec/mmlogic.proto.
Most of the documentation for what these calls should do is in that file!

Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

*/

package set

// Intersection returns the interection of two sets.
func Intersection(a []string, b []string) (out []string) {

	hash := make(map[string]bool)

	for _, v := range a {
		hash[v] = true
	}

	for _, v := range b {
		if _, found := hash[v]; found {
			out = append(out, v)
		}
	}

	return out

}

// Union returns the union of two sets.
func Union(a []string, b []string) (out []string) {

	hash := make(map[string]bool)

	// collect all values from input args
	for _, v := range a {
		hash[v] = true
	}

	for _, v := range b {
		hash[v] = true
	}

	// put values into string array
	for k := range hash {
		out = append(out, k)
	}

	return out

}

// Difference returns the items in the first argument that are not in the
// second (set 'a' - set 'b')
func Difference(a []string, b []string) (out []string) {

	hash := make(map[string]bool)
	out = append([]string{}, a...)

	for _, v := range b {
		hash[v] = true
	}

	// Iterate through output, removing items found in b
	for i := 0; i < len(out); {
		if _, found := hash[out[i]]; found {
			// Remove this element by moving the copying the last element of the
			// array to this index and then slicing off the last element.
			// https://stackoverflow.com/a/37335777/3113674
			out[i] = out[len(out)-1]
			out = out[:len(out)-1]
		} else {
			i++
		}
	}

	return out
}
