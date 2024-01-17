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

// Package lexparse defines a set of interfaces that can be used to define
// generic lexers and parsers over byte streams.
package lexparse

import "context"

// LexParse runs the Lexer passing lexemes to the parser functions.
func LexParse[V comparable](
	ctx context.Context,
	r BufferedRuneReader,
	initState State,
	initFn ParseFn[V],
) (*Tree[V], error) {
	l := NewLexer(r, initState)
	p := NewParser[V](l.Lex(ctx))
	return p.Parse(ctx, initFn)
}
