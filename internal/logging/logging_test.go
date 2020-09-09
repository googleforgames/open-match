// Copyright 2019 Google LLC
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

package logging

import (
	"fmt"
	"reflect"
	"testing"

	stackdriver "github.com/TV4/logrus-stackdriver-formatter"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestNewFormatter(t *testing.T) {
	testCases := []struct {
		in       string
		expected interface{}
	}{
		{"", &logrus.TextFormatter{}},
		{"json", &logrus.JSONFormatter{}},
		{"stackdriver", stackdriver.NewFormatter()},
		{"text", &logrus.TextFormatter{}},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("newFormatter(%s) => %s", tc.in, tc.expected), func(t *testing.T) {
			require := require.New(t)
			actual := newFormatter(tc.in)
			require.Equal(reflect.TypeOf(tc.expected), reflect.TypeOf(actual))
		})
	}
}

func TestIsDebugLevel(t *testing.T) {
	testCases := []struct {
		in       logrus.Level
		expected bool
	}{
		{logrus.TraceLevel, true},
		{logrus.DebugLevel, true},
		{logrus.InfoLevel, false},
		{logrus.WarnLevel, false},
		{logrus.ErrorLevel, false},
		{logrus.FatalLevel, false},
		{logrus.PanicLevel, false},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("isDebugLevel(%s) => %t", tc.in, tc.expected), func(t *testing.T) {
			require := require.New(t)
			actual := isDebugLevel(tc.in)
			require.Equal(tc.expected, actual)
		})
	}
}

func TestToLevel(t *testing.T) {
	testCases := []struct {
		in       string
		expected logrus.Level
	}{
		{"trace", logrus.TraceLevel},
		{"debug", logrus.DebugLevel},
		{"info", logrus.InfoLevel},
		{"warn", logrus.WarnLevel},
		{"warning", logrus.WarnLevel},
		{"error", logrus.ErrorLevel},
		{"fatal", logrus.FatalLevel},
		{"panic", logrus.PanicLevel},
		{"info", logrus.InfoLevel},
		{"", logrus.InfoLevel},
		{"nothing", logrus.InfoLevel},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(fmt.Sprintf("toLevel(%s) => %s", tc.in, tc.expected), func(t *testing.T) {
			require := require.New(t)
			actual := toLevel(tc.in)
			require.Equal(tc.expected, actual)
		})
	}
}
