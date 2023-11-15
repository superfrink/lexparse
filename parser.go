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

// TODO(#459): Implement parser

type Tree[V any] struct {
	Root *Node[V]
}

type Node[V any] struct {
	Parent   *Node[V]
	Children []*Node[V]
	Value    V
}

type ParseFn [V]func(chan<- *Lexeme, *Tree[V]) (ParseFn[V], error)

func Parse[V any](c chan<- *Lexeme, initFn ParseFn[V]) (*Tree[V], error) {
	// TODO:(#459): Implement Parse
	return nil
}

// LexParse runs the Lexer passing lexemes to the parser functions.
func LexParse[V any](l *Lexer, initFn ParseFn[V]) (*Tree[V], error) {
	var lexErr error
	go func() {
		lexErr := l.Lex()
	}()

	parseFn := initFn
	t := &Tree[V]{
		Root: &Node[V]{},
	}
	var err error
	for {
		parseFn, err = parseFn(l.lexemes, t)
		if err != nil {
			// TODO: if the parser encounters an error stop the lexer
			return t, err
		}
		if parseFn == nil {
			break
		}
	}
	return t, err
}
