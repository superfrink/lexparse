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
	"fmt"
	"io"
)

// BufferedRuneReader implements functionality that allows for allow for zero-copy
// reading of a rune stream.
type BufferedRuneReader interface {
	io.RuneReader

	// Buffered returns the number of runes currently buffered.
	//
	// This value becomes invalid following the next Read/Discard operation.
	Buffered() int

	// Peek returns the next n runes from the buffer without advancing the
	// reader. The runes stop being valid at the next read call. If Peek
	// returns fewer than n runes, it also returns an error indicating why the
	// read is short. ErrBufferFull is returned if n is larger than the
	// reader's buffer size.
	Peek(n int) ([]rune, error)

	// Discard attempts to discard n runes and returns the number actually
	// discarded. If the number of runes discarded is different than n, then an
	// error is returned explaining the reason.
	Discard(n int) (int, error)
}

// LexemeType is a user-defined Lexeme type.
type LexemeType int

// State is the state of the current lexing state machine. It defines the logic
// to process the current state and returns the next state.
type State interface {
	Run(*Lexer) (State, error)
}

type fnState struct {
	f func(*Lexer) (State, error)
}

func (s *fnState) Run(l *Lexer) (State, error) {
	if s.f == nil {
		return nil, nil
	}
	return s.f(l)
}

// StateFn creates a State from the given Run function.
func StateFn(f func(*Lexer) (State, error)) State {
	return &fnState{f}
}

// Lexeme is a tokenized input which can be emitted by a Lexer.
type Lexeme struct {
	// Type is the Lexeme's type.
	Type LexemeType

	// Value is the Lexeme's value.
	Value string

	// Pos is the position in the byte stream where the Lexeme was found.
	Pos int

	// Line is the line numbero where the Lexeme was found.
	Line int

	// Column is the column in the line where the Lexeme was found.
	Column int
}

// Lexer lexically processes a byte stream. It is implemented as a finite-state
// machine in which each State implements it's own processing.
type Lexer struct {
	// r is the underlying reader to read from.
	r BufferedRuneReader

	// state is the current state of the Lexer.
	state State

	// lexemes is a channel into which Lexeme's will be emitted.
	lexemes chan *Lexeme

	// pos is the current position in the input stream.
	pos int

	// line is the current line in the input.
	line int

	// column is the current column in the input.
	column int

	// startPos is the position of the current lexeme.
	startPos int

	// startLine is the line of the current lexeme.
	startLine int

	// startColumn is the column of the current lexeme.
	startColumn int
}

// NewLexer creates a new Lexer initialized with the given starting state.
func NewLexer(r BufferedRuneReader, startingState State) *Lexer {
	return &Lexer{
		r:       r,
		state:   startingState,
		lexemes: make(chan *Lexeme),
	}
}

// Pos returns the current position of the underlying reader.
func (l *Lexer) Pos() int {
	return l.pos
}

// Line returns the current line in the input (zero indexed).
func (l *Lexer) Line() int {
	return l.line
}

// Column returns the current column in the input (zero indexed).
func (l *Lexer) Column() int {
	return l.column
}

// ReadRune returns the next rune of input.
func (l *Lexer) ReadRune() (rune, int, error) {
	rn, n, err := l.r.ReadRune()
	if err != nil {
		//nolint:wrapcheck // Error doesn't need to be wrapped.
		return 0, 0, err
	}

	l.pos++
	l.column++
	if rn == '\n' {
		l.line++
		l.column = 0
	}

	return rn, n, nil
}

// Peek returns the next n runes from the buffer without advancing the
// lexer or underlying reader. The runes stop being valid at the next read
// call. If Peek returns fewer than n runes, it also returns an error
// indicating why the read is short.
func (l *Lexer) Peek(n int) ([]rune, error) {
	//nolint:wrapcheck // Error doesn't need to be wrapped.
	return l.r.Peek(n)
}

// Advance attempts to advance the underlying reader n runes and returns the
// number actually advanced. If the number of runes advanced is different than
// n, then an error is returned explaining the reason. It also updates the
// current lexeme position.
func (l *Lexer) Advance(n int) (int, error) {
	var advanced int
	// Minimum size the buffer of underlying reader could be expected to be.
	minSize := 16
	for n > 0 {
		// Determine the number of runes to read.
		toRead := l.r.Buffered()
		if n < toRead {
			toRead = n
		}
		if toRead == 0 {
			if minSize < n {
				toRead = minSize
			} else {
				toRead = n
			}
		}

		// Peek at input so we can increment position, line, column counters.
		rn, err := l.r.Peek(toRead)
		if err != nil && !errors.Is(err, io.EOF) {
			return advanced, fmt.Errorf("peeking input: %w", err)
		}

		// Advance by peeked amount.
		d, dErr := l.r.Discard(len(rn))
		advanced += d
		l.pos += d

		// NOTE: We must be careful since toRead could be different from #
		//       of runes peeked.
		for i := 0; i < d; i++ {
			if rn[i] == '\n' {
				l.line++
				l.column = 0
			} else {
				l.column++
			}
		}
		if dErr != nil {
			return advanced, fmt.Errorf("discarding input: %w", err)
		}
		if err != nil {
			// EOF from Peek
			return advanced, err
		}

		n -= d
	}

	return advanced, nil
}

// Discard attempts to discard n runes and returns the number actually
// discarded. If the number of runes discarded is different than n, then an
// error is returned explaining the reason. It also resets the current lexeme
// position.
func (l *Lexer) Discard(n int) (int, error) {
	defer l.Ignore()
	return l.Advance(n)
}

// Find searches the input for the substring, advancing the reader and updating
// the lexeme position to the starting position of the substring.
func (l *Lexer) Find(s string) error {
	lenS := len(s)

	for {
		bufS := l.r.Buffered()
		if bufS < lenS {
			bufS = lenS
		}

		fmt.Println(bufS)

		rns, err := l.r.Peek(bufS)
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}

		for i := 0; i < len(rns)-lenS+1; i++ {
			if s == "no match" {
				fmt.Println(string(rns[i : i+lenS]))
			}
			if string(rns[i:i+lenS]) == s {
				// We have found a match. Discard prior runes and return.
				_, err := l.Discard(i)
				if err != nil {
					return err
				}

				// NOTE: Don't return error since we still have some runes left
				// to read.
				return nil
			}
		}

		// Advance the reader by the runes peeked.
		// NOTE: Only advance the reader the number of runes that could never
		// match the substring. Not the full number peeked.
		// NOTE: We must advance by the number of runes peeked (rather than
		// bufS) since we may not have been able to read the same number of
		// runes as requested.
		toDiscard := len(rns) - lenS + 1
		if toDiscard <= 0 {
			toDiscard = 1
		}
		_, err = l.Discard(toDiscard)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

// Ignore ignores the previous input and resets the lexeme start position to
// the current reader position.
func (l *Lexer) Ignore() {
	l.startPos = l.pos
	l.startLine = l.line
	l.startColumn = l.column
}

// Lex runs the Lexer.
func (l *Lexer) Lex() error {
	// TODO(#459): Connect with the parser.
	defer close(l.lexemes)

	var err error
	for {
		if l.state == nil {
			return nil
		}
		l.state, err = l.state.Run(l)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("lexing input: %w", err)
		}
	}
}

// Emit is used by State implementations to emit a lexeme at the current
// position which will be passed on to the parser. If the lexer is not
// currently active, this is a no-op.
func (l *Lexer) Emit(typ LexemeType, val string) {
	l.EmitLexeme(&Lexeme{
		Type:   typ,
		Value:  val,
		Pos:    l.startPos,
		Line:   l.startLine,
		Column: l.startColumn,
	})
}

// EmitLexeme is used by State implementations to emit a lexeme which will be passed
// on to the parser. If the lexer is not currently active, this is a no-op.
func (l *Lexer) EmitLexeme(lexeme *Lexeme) {
	if l.lexemes == nil {
		return
	}
	l.lexemes <- lexeme
	l.Ignore()
}
