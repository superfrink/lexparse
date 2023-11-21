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

func TestLexer_Peek(t *testing.T) {
	t.Parallel()

	l := NewLexer(runeio.NewReader(strings.NewReader("Hello\nWorld!")), &wordState{})

	rns, err := l.Peek(6)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got, want := string(rns), "Hello\n"; got != want {
		t.Errorf("Peek: want: %q, got: %q", want, got)
	}

	rns, err = l.Peek(16)
	if !errors.Is(err, io.EOF) {
		t.Errorf("unexpected error: %v", err)
	}
	if got, want := string(rns), "Hello\nWorld!"; got != want {
		t.Errorf("Peek: want: %q, got: %q", want, got)
	}

	if got, want := l.Pos(), 0; got != want {
		t.Errorf("Pos: want: %v, got: %v", want, got)
	}

	if got, want := l.Line(), 0; got != want {
		t.Errorf("Line: want: %v, got: %v", want, got)
	}

	if got, want := l.Column(), 0; got != want {
		t.Errorf("Column: want: %v, got: %v", want, got)
	}

	if got, want := l.startPos, 0; got != want {
		t.Errorf("startPos: want: %v, got: %v", want, got)
	}

	if got, want := l.startLine, 0; got != want {
		t.Errorf("startLine: want: %v, got: %v", want, got)
	}

	if got, want := l.startColumn, 0; got != want {
		t.Errorf("startColumn: want: %v, got: %v", want, got)
	}
}

func TestLexer_Advance(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		l := NewLexer(runeio.NewReader(strings.NewReader("Hello\n!World!")), &wordState{})

		advanced, err := l.Advance(7)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got, want := advanced, 7; got != want {
			t.Errorf("Advance: want: %v, got: %v", want, got)
		}

		rns, err := l.Peek(6)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got, want := string(rns), "World!"; got != want {
			t.Errorf("Peek: want: %q, got: %q", want, got)
		}

		if got, want := l.Pos(), 7; got != want {
			t.Errorf("Pos: want: %v, got: %v", want, got)
		}

		if got, want := l.Line(), 1; got != want {
			t.Errorf("Line: want: %v, got: %v", want, got)
		}

		if got, want := l.Column(), 1; got != want {
			t.Errorf("Column: want: %v, got: %v", want, got)
		}

		if got, want := l.startPos, 0; got != want {
			t.Errorf("startPos: want: %v, got: %v", want, got)
		}

		if got, want := l.startLine, 0; got != want {
			t.Errorf("startLine: want: %v, got: %v", want, got)
		}

		if got, want := l.startColumn, 0; got != want {
			t.Errorf("startColumn: want: %v, got: %v", want, got)
		}
	})

	t.Run("past end", func(t *testing.T) {
		l := NewLexer(runeio.NewReader(strings.NewReader("Hello\n!World!")), &wordState{})

		advanced, err := l.Advance(14)
		if !errors.Is(err, io.EOF) {
			t.Errorf("unexpected error: %v", err)
		}
		if got, want := advanced, 13; got != want {
			t.Errorf("Advance: want: %v, got: %v", want, got)
		}

		if got, want := l.Pos(), 13; got != want {
			t.Errorf("Pos: want: %v, got: %v", want, got)
		}

		if got, want := l.Line(), 1; got != want {
			t.Errorf("Line: want: %v, got: %v", want, got)
		}

		if got, want := l.Column(), 7; got != want {
			t.Errorf("Column: want: %v, got: %v", want, got)
		}

		if got, want := l.startPos, 0; got != want {
			t.Errorf("startPos: want: %v, got: %v", want, got)
		}

		if got, want := l.startLine, 0; got != want {
			t.Errorf("startLine: want: %v, got: %v", want, got)
		}

		if got, want := l.startColumn, 0; got != want {
			t.Errorf("startColumn: want: %v, got: %v", want, got)
		}
	})
}

func TestLexer_Discard(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		l := NewLexer(runeio.NewReader(strings.NewReader("Hello\n!World!")), &wordState{})

		discarded, err := l.Discard(7)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got, want := discarded, 7; got != want {
			t.Errorf("Discard: want: %v, got: %v", want, got)
		}

		rns, err := l.Peek(6)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got, want := string(rns), "World!"; got != want {
			t.Errorf("Peek: want: %q, got: %q", want, got)
		}

		if got, want := l.Pos(), 7; got != want {
			t.Errorf("Pos: want: %v, got: %v", want, got)
		}

		if got, want := l.Line(), 1; got != want {
			t.Errorf("Line: want: %v, got: %v", want, got)
		}

		if got, want := l.Column(), 1; got != want {
			t.Errorf("Column: want: %v, got: %v", want, got)
		}

		if got, want := l.startPos, 7; got != want {
			t.Errorf("startPos: want: %v, got: %v", want, got)
		}

		if got, want := l.startLine, 1; got != want {
			t.Errorf("startLine: want: %v, got: %v", want, got)
		}

		if got, want := l.startColumn, 1; got != want {
			t.Errorf("startColumn: want: %v, got: %v", want, got)
		}
	})

	t.Run("past end", func(t *testing.T) {
		l := NewLexer(runeio.NewReader(strings.NewReader("Hello\n!World!")), &wordState{})

		discarded, err := l.Discard(14)
		if !errors.Is(err, io.EOF) {
			t.Errorf("unexpected error: %v", err)
		}
		if got, want := discarded, 13; got != want {
			t.Errorf("Discard: want: %v, got: %v", want, got)
		}

		if got, want := l.Pos(), 13; got != want {
			t.Errorf("Pos: want: %v, got: %v", want, got)
		}

		if got, want := l.Line(), 1; got != want {
			t.Errorf("Line: want: %v, got: %v", want, got)
		}

		if got, want := l.Column(), 7; got != want {
			t.Errorf("Column: want: %v, got: %v", want, got)
		}

		if got, want := l.startPos, 13; got != want {
			t.Errorf("startPos: want: %v, got: %v", want, got)
		}

		if got, want := l.startLine, 1; got != want {
			t.Errorf("startLine: want: %v, got: %v", want, got)
		}

		if got, want := l.startColumn, 7; got != want {
			t.Errorf("startColumn: want: %v, got: %v", want, got)
		}
	})
}

func TestLexer_Find(t *testing.T) {
	t.Parallel()

	t.Run("match", func(t *testing.T) {
		l := NewLexer(runeio.NewReader(strings.NewReader("Hello\n!World!")), &wordState{})

		err := l.Find("World")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		rns, err := l.Peek(6)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got, want := string(rns), "World!"; got != want {
			t.Errorf("Peek: want: %q, got: %q", want, got)
		}

		if got, want := l.Pos(), 7; got != want {
			t.Errorf("Pos: want: %v, got: %v", want, got)
		}

		if got, want := l.Line(), 1; got != want {
			t.Errorf("Line: want: %v, got: %v", want, got)
		}

		if got, want := l.Column(), 1; got != want {
			t.Errorf("Column: want: %v, got: %v", want, got)
		}

		if got, want := l.startPos, 7; got != want {
			t.Errorf("startPos: want: %v, got: %v", want, got)
		}

		if got, want := l.startLine, 1; got != want {
			t.Errorf("startLine: want: %v, got: %v", want, got)
		}

		if got, want := l.startColumn, 1; got != want {
			t.Errorf("startColumn: want: %v, got: %v", want, got)
		}
	})

	t.Run("no match", func(t *testing.T) {
		l := NewLexer(runeio.NewReader(strings.NewReader("Hello\n!World!")), &wordState{})

		err := l.Find("no match")
		if !errors.Is(err, io.EOF) {
			t.Errorf("unexpected error: %v", err)
		}

		if got, want := l.Pos(), 13; got != want {
			t.Errorf("Pos: want: %v, got: %v", want, got)
		}

		if got, want := l.Line(), 1; got != want {
			t.Errorf("Line: want: %v, got: %v", want, got)
		}

		if got, want := l.Column(), 7; got != want {
			t.Errorf("Column: want: %v, got: %v", want, got)
		}

		if got, want := l.startPos, 13; got != want {
			t.Errorf("startPos: want: %v, got: %v", want, got)
		}

		if got, want := l.startLine, 1; got != want {
			t.Errorf("startLine: want: %v, got: %v", want, got)
		}

		if got, want := l.startColumn, 7; got != want {
			t.Errorf("startColumn: want: %v, got: %v", want, got)
		}
	})
}

func TestLexer_lexemes(t *testing.T) {
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
