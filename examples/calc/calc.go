package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
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

var (
	errUnexpectedToken = errors.New("unexpected token")
	errMissingToken    = errors.New("missing token")
)

func (w *lexState) Run(_ context.Context, l *lexparse.Lexer) (lexparse.State, error) {
	w.CurrentToken = noToken

	for {
		// fmt.Printf("cur token: %d\n", w.CurrentToken)

		rn, _, err := l.ReadRune()
		// fmt.Printf("rn: %q\n", rn)

		// TODO: remove need for spaces between lexemes
		//nolint:gocritic // ignore ifElseChain because switching on more than one variable
		if rn == ' ' && w.CurrentToken == noToken {
			// eat the space
			l.Ignore()
		} else if rn == ' ' || errors.Is(err, io.EOF) {
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
					// FIXME: Parser does not see these errors because it is
					//        reading lexemes from a channel and does not have
					//        the Lexer to call Lexer.Err() and see this error
					return nil, fmt.Errorf("%q found in mulOp, %w", rn, errUnexpectedToken)
				}
				w.CurrentToken = mulOpToken

			case '+', '-':
				if w.CurrentToken != noToken {
					return nil, fmt.Errorf("%q found in addOp %w", rn, errUnexpectedToken)
				}
				w.CurrentToken = addOpToken

			case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
				if w.CurrentToken != noToken && w.CurrentToken != natNumberToken {
					return nil, fmt.Errorf("%q found in number %w", rn, errUnexpectedToken)
				}
				w.CurrentToken = natNumberToken
			}
		}
		if err != nil {
			return nil, fmt.Errorf("reading rune: %w", err)
		}
	}
}

// printTreeNodes walks tree nodes and prints a visualization of the tree.
func printTreeNodes[T any](n int, node *lexparse.Node[T]) {
	log.Printf(strings.Repeat(" ", n)+"[%T] Value: [%T]  %+v", node, node.Value, node.Value)

	for _, c := range node.Children {
		printTreeNodes[T](n+1, c)
	}
}

func myParseFn(p *lexparse.Parser[calcToken]) func(
	context.Context, *lexparse.Parser[calcToken],
) (
	lexparse.ParseFn[calcToken], error,
) {
	return func(_ context.Context, _ *lexparse.Parser[calcToken]) (lexparse.ParseFn[calcToken], error) {
		for {
			// printTreeNodes(0, p.Tree().Root)

			lexeme := p.Next()
			if lexeme == nil {
				break
			}
			// fmt.Printf("lexeme: %+v\n", lexeme)
			token := calcToken{
				Type:  lexeme.Type,
				Value: lexeme.Value,
			}
			switch lexeme.Type {
			case mulOpToken:
				nextLexeme := p.Next()
				if nextLexeme == nil {
					return nil, fmt.Errorf(
						"nothing found after mulOp: %w",
						errMissingToken,
					)
				}
				if nextLexeme.Type != natNumberToken {
					return nil, fmt.Errorf(
						"number not found after mulOp: %q, %w",
						nextLexeme.Value,
						errUnexpectedToken,
					)
				}

				nextToken := calcToken{
					Type:  nextLexeme.Type,
					Value: nextLexeme.Value,
				}

				switch p.Pos().Value.Type {
				case natNumberToken, mulOpToken:
					p.Push(token)
					p.RotateLeft()
					p.Node(nextToken)

				case addOpToken:
					p.Push(token)
					p.AdoptSibling()
					p.Node(nextToken)

				default:
					return nil, fmt.Errorf(
						"number not found before mulOp: %q, %w",
						p.Pos().Value.Value,
						errUnexpectedToken,
					)
				}

			case addOpToken:
				nextLexeme := p.Next()
				if nextLexeme == nil {
					return nil, fmt.Errorf(
						"nothing found after addOp: %w",
						errMissingToken,
					)
				}
				if nextLexeme.Type != natNumberToken {
					return nil, fmt.Errorf(
						"number not found after addOp: %q, %w",
						nextLexeme.Value,
						errUnexpectedToken,
					)
				}

				nextToken := calcToken{
					Type:  nextLexeme.Type,
					Value: nextLexeme.Value,
				}

				switch p.Pos().Value.Type {
				case natNumberToken, mulOpToken, addOpToken:
					p.Push(token)
					p.RotateLeft()
					p.Node(nextToken)

				default:
					return nil, fmt.Errorf(
						"number not found before addOp: %q, %w",
						p.Pos().Value.Value,
						errUnexpectedToken,
					)
				}

			case natNumberToken:
				p.Push(token)
			}
		}
		return nil, nil
	}
}

// calculate performs the calulation represented by the parse tree.
func calculate(tree *lexparse.Tree[calcToken]) (int, error) {
	return doCalculate(tree.Root.Children[0])
}

// doCalculate is a recursive helper function used by calculate.
func doCalculate(n *lexparse.Node[calcToken]) (int, error) {
	switch n.Value.Type {
	case mulOpToken, addOpToken:

		if len(n.Children) != 2 {
			return 0, fmt.Errorf(
				"expecting 2 node children in calculation: %q %d %w",
				n.Value.Value,
				len(n.Children),
				errUnexpectedToken,
			)
		}

		r1, err1 := doCalculate(n.Children[0])
		if err1 != nil {
			return 0, err1
		}
		r2, err2 := doCalculate(n.Children[1])
		if err2 != nil {
			return 0, err2
		}

		switch n.Value.Value {
		case "+":
			return r1 + r2, nil
		case "-":
			return r1 - r2, nil
		case "*":
			return r1 * r2, nil
		case "/":
			return r1 / r2, nil
		}

	case natNumberToken:
		result, err := strconv.Atoi(n.Value.Value)
		if err != nil {
			return 0, fmt.Errorf("atoi failed: %q %w", n.Value.Value, err)
		}
		return result, nil
	}

	return 0, fmt.Errorf("unknown token type in calculation: %q %w", n.Value, errUnexpectedToken)
}

func main() {
	inReader := bufio.NewReader(os.Stdin)

	l := lexparse.NewLexer(runeio.NewReader(inReader), &lexState{})
	lexemes := l.Lex(context.Background())

	p := lexparse.NewParser[calcToken](lexemes)
	pFn := myParseFn(p)

	ctx := context.Background()
	tree, err := p.Parse(ctx, pFn)
	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}

	fmt.Printf("\ntree: %+v\n", tree)
	printTreeNodes(0, tree.Root)

	result, err := calculate(tree)
	if err != nil {
		log.Fatalf("calculate failed.  %s", err)
	}

	fmt.Println(result)
}
