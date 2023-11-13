// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lexparse

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ianlewis/runeio"
)

const (
	unusedType LexemeType = iota
	wordType
)

type wordState struct{}

func (w *wordState) Run(l *Lexer) (State, error) {
	var word []rune
	for {
		rn, _, err := l.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if len(word) > 0 {
					l.Emit(wordType, string(word))
				}
				return nil, nil
			}
			return nil, err
		}
		if rn == ' ' {
			if len(word) > 0 {
				l.Emit(wordType, string(word))
			}
			return w, nil
		}

		word = append(word, rn)
	}
}

func TestLexer(t *testing.T) {
	t.Parallel()

	l := NewLexer(runeio.NewReader(strings.NewReader("Hello World!")), &wordState{})
	go func() {
		if err := l.Lex(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}()

	var items []*Lexeme
	for item := range l.lexemes {
		items = append(items, item)
	}
	got := items
	want := []*Lexeme{
		{
			Type:   wordType,
			Value:  "Hello",
			Pos:    0,
			Line:   0,
			Column: 0,
		},
		{
			Type:   wordType,
			Value:  "World!",
			Pos:    6,
			Line:   0,
			Column: 6,
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected output (-want +got):\n%s", diff)
	}
}
