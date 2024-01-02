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
	"context"
	"errors"
	"io"
)

// TODO(#459): Implement parser

type Tree[V any] struct {
	Root *Node[V]
}

type Node[V any] struct {
	Parent   *Node[V]
	Children []*Node[V]
	Value    V
	// TODO: Position,Line,Column in original input.
}

// FIXME: Remove channel
type ParseFn[V any] func(context.Context, *Parser[V]) (ParseFn[V], error)

func NewParser[V any](lexemes <-chan *Lexeme) *Parser[V] {
	root := &Node[V]{}
	p := &Parser[V]{
		lexemes: lexemes,
		tree: &Tree[V]{
			Root: root,
		},
		node: root,
	}
	return p
}

type Parser[V any] struct {
	lexemes <-chan *Lexeme

	tree *Tree[V]
	// node is the current node under processing.
	node *Node[V]

	// lexeme is the next lexeme in the stream.
	lexeme *Lexeme
}

func (p *Parser[V]) Parse(ctx context.Context, parseFn ParseFn[V]) (*Tree[V], error) {
	for {
		if parseFn == nil {
			break
		}
		select {
		case <-ctx.Done():
			return p.Tree(), ctx.Err()
		default:
		}

		var err error
		parseFn, err = parseFn(ctx, p)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return p.Tree(), err
		}
	}
	return p.Tree(), nil
}

func (p *Parser[V]) Tree() *Tree[V] {
	return p.tree
}

// Peek returns the next Lexeme from the lexer without consuming it.
func (p *Parser[V]) Peek() *Lexeme {
	if p.lexeme != nil {
		return p.lexeme
	}
	l, ok := <-p.lexemes
	if !ok {
		return nil
	}
	p.lexeme = l
	return p.lexeme
}

// Next returns the next Lexeme from the lexer.
func (p *Parser[V]) Next() *Lexeme {
	l := p.Peek()
	p.lexeme = nil
	return l
}

// Pos returns the current node position in the tree.
func (p *Parser[V]) Pos() *Node[V] {
	return p.node
}

// Push creates a new node, adds it as a child to the current node, and sets it
// as the current node.
func (p *Parser[V]) Push(v V) *Node[V] {
	n := p.Node(v)
	p.node = n
	return n
}

// Node creates a new node and adds it as a child to the current node.
func (p *Parser[V]) Node(v V) *Node[V] {
	cur := p.Pos()
	node := &Node[V]{
		Parent: p.Pos(),
		Value:  v,
	}
	cur.Children = append(cur.Children, node)
	return node
}

// Pop updates the current node position to the current node's parent
// returning the previous current node.
func (p *Parser[V]) Pop() *Node[V] {
	n := p.node
	p.node = p.node.Parent
	return n
}

// RotateLeft moves the current node to the position it's parent had,
// making the parent into the current node's child.
func (p *Parser[V]) RotateLeft() *Node[V] {
	// n = node , op = original parent , gp = grand parent
	n := p.node
	op := n.Parent
	gp := op.Parent

	// fmt.Printf("before\nn:  %+v\nop: %+v\ngp: %+v\n", n, op, gp)

	// Remove n from op's Children
	opChildren := op.Children[:0]
	for _, x := range op.Children {
		if x != n {
			opChildren = append(opChildren, x)
		}
	}
	op.Children = opChildren

	// Add op to n's Children
	n.Children = append(n.Children, op)

	// Update op's Parent
	op.Parent = n

	// Update n's Parent
	n.Parent = gp

	// Replace op with n in gp's Children
	gpChildren := gp.Children[:0]
	for _, x := range gp.Children {
		if x != op {
			gpChildren = append(gpChildren, x)
		} else {
			gpChildren = append(gpChildren, n)
		}
	}
	gp.Children = gpChildren

	// fmt.Printf("after\nn:  %+v\nop: %+v\ngp: %+v\n", n, op, gp)
	return n
}

// AdoptSibling moves the current node's previous sibling into the node's child.
func (p *Parser[V]) AdoptSibling() *Node[V] {

	// n = node , op = original parent , s = sibling
	n := p.node
	op := n.Parent
	var s *Node[V]

	// Find the previous sibling
	for _, x := range op.Children {
		if x == n {
			break
		}
		s = x
	}
	// fmt.Printf("before\nn:  %+v\nop: %+v\ns: %+v\n", n, op, s)

	// Remove s from op's Children
	opChildren := op.Children[:0]
	for _, x := range op.Children {
		if x != s {
			opChildren = append(opChildren, x)
		}
	}
	op.Children = opChildren

	// Add s to n's Children
	n.Children = append(n.Children, s)

	// Update s's Parent
	s.Parent = n

	// fmt.Printf("after\nn:  %+v\nop: %+v\ns: %+v\n", n, op, s)

	return n
}
