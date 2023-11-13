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

// Line returns the current line in the input.
func (l *Lexer) Line() int {
	return l.line
}

// Column returns the current column in the input.
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

// Discard attempts to discard n runes and returns the number actually
// discarded. If the number of runes discarded is different than n, then an
// error is returned explaining the reason. It also resets the current lexeme
// position.
func (l *Lexer) Discard(n int) (int, error) {
	var discarded int
	defer l.Ignore()
	// TODO(github.com/ianlewis/runeio/issues/51): Optimize using Buffered method.
	// Minimum size the buffer of underlying reader could be expected to be.
	minSize := 16
	for n > 0 {
		toRead := minSize
		if n < minSize {
			toRead = n
		}

		// Peek at input so we can increment position, line, column counters.
		rn, err := l.r.Peek(toRead)
		if err != nil {
			return discarded, fmt.Errorf("peeking input: %w", err)
		}

		d, err := l.r.Discard(toRead)
		discarded += d
		l.pos += d
		l.column += d

		// NOTE: We must be careful since # discarded could be different from #
		//       of runes peeked.
		for i := 0; i < d; i++ {
			if rn[i] == '\n' {
				l.line++
				l.column = 0
			}
		}
		if err != nil {
			return discarded, fmt.Errorf("discarding input: %w", err)
		}
		n -= d
	}

	return discarded, nil
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
