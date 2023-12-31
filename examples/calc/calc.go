package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/ianlewis/lexparse"
	"github.com/ianlewis/runeio"
)

// Grammar:
//
//   exp ->  exp addOp exp | term
//   addOp -> '+' | '-'
//   term -> term mulop term | factor
//   mulOp -> '*' | '/'
//   factor -> number

const (
	noToken = iota
	mulOpToken
	addOpToken
	natNumberToken
)

type calcToken struct {
	Type  lexparse.LexemeType
	Value string
}

type lexState struct {
	CurrentToken lexparse.LexemeType
}

func (w *lexState) Run(_ context.Context, l *lexparse.Lexer) (lexparse.State, error) {
	w.CurrentToken = noToken

	for {
		rn, _, err := l.ReadRune()
		// TODO: remove need for spaces between lexemes
		if rn == ' ' || errors.Is(err, io.EOF) {
			word := l.Lexeme(w.CurrentToken)
			word.Value = strings.TrimRight(word.Value, " ")
			if word.Value != "" {
				l.Emit(word)
			}
			w.CurrentToken = noToken

		} else {
			switch rn {
			case '*', '/':
				if w.CurrentToken != noToken {
					return nil, fmt.Errorf("unexpected %q in input", rn)
				}
				w.CurrentToken = mulOpToken

			case '+', '-':
				if w.CurrentToken != noToken {
					return nil, fmt.Errorf("unexpected %q in input", rn)
				}
				w.CurrentToken = addOpToken

			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				if w.CurrentToken != noToken && w.CurrentToken != natNumberToken {
					return nil, fmt.Errorf("unexpected %q in input", rn)
				}
				w.CurrentToken = natNumberToken
			}
		}
		if err != nil {
			return nil, err
		}

	}
}

// printTreeNodes walks tree nodes and prints a visualization of the tree.
func printTreeNodes[T any](n int, node *lexparse.Node[T]) {
	log.Printf(strings.Repeat(" ", n)+"Value: %+v", node.Value)

	for _, c := range node.Children {
		printTreeNodes[T](n+1, c)
	}
}

func main() {

	// l := lexparse.NewLexer(runeio.NewReader(strings.NewReader("10 + 2 * 3")), &lexState{})
	// l := lexparse.NewLexer(runeio.NewReader(strings.NewReader("1 + 2 + 3")), &lexState{})
	// l := lexparse.NewLexer(runeio.NewReader(strings.NewReader("1 * 2 * 3")), &lexState{})
	l := lexparse.NewLexer(runeio.NewReader(strings.NewReader("1 + 2 * 3")), &lexState{})
	lexemes := l.Lex(context.Background())
	fmt.Printf("lexemes: %+t\n", lexemes)

	p := lexparse.NewParser[calcToken](lexemes)
	stack := []calcToken{}
	myParseFn := func(_ context.Context, pFn *lexparse.Parser[calcToken]) (lexparse.ParseFn[calcToken], error) {
		for {
			fmt.Printf("\nstack: %+v\n", stack)
			printTreeNodes(0, p.Tree().Root)

			lexeme := p.Next()
			if lexeme == nil {
				break
			}
			fmt.Printf("lexeme: %+v\n", lexeme)
			token := calcToken{
				Type:  lexeme.Type,
				Value: lexeme.Value,
			}
			switch lexeme.Type {
			case mulOpToken:
				nextLexeme := p.Next()
				if nextLexeme.Type != natNumberToken {
					return nil, fmt.Errorf("number not found after mulOp: %+v", nextLexeme)
				}

				nextToken := calcToken{
					Type:  nextLexeme.Type,
					Value: nextLexeme.Value,
				}

				var prevToken calcToken
				if len(stack) > 0 {
					prevToken, stack = stack[0], stack[1:]
					if prevToken.Type != natNumberToken {
						return nil, fmt.Errorf("number not found before mulOp: %+v", prevToken)
					}

					p.Push(token)
					p.Node(prevToken)
					p.Node(nextToken)

				} else {
					p.Push(token)
					p.AdoptSibling()
					p.Node(nextToken)
				}

			case addOpToken:
				nextLexeme := p.Next()
				if nextLexeme.Type != natNumberToken {
					return nil, fmt.Errorf("number not found after addOp: %+v", nextLexeme)
				}

				nextToken := calcToken{
					Type:  nextLexeme.Type,
					Value: nextLexeme.Value,
				}

				var prevToken calcToken
				if len(stack) > 0 {
					prevToken, stack = stack[0], stack[1:]
					if prevToken.Type != natNumberToken {
						return nil, fmt.Errorf("number not found before addOp: %+v", prevToken)
					}

					p.Push(token)
					p.Node(prevToken)
					p.Node(nextToken)

				} else {
					p.Push(token)
					p.RotateLeft()
					p.Node(nextToken)
				}

			case natNumberToken:
				stack = append(stack, token)
			}
		}
		return nil, nil
	}

	ctx := context.Background()
	tree, err := p.Parse(ctx, myParseFn)
	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}

	fmt.Printf("\ntree: %+v\n", tree)
	printTreeNodes(0, tree.Root)
}
