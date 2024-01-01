package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"testing"

	"github.com/ianlewis/lexparse"
	"github.com/ianlewis/runeio"
)

var (
	errTreeMismatchSize  = errors.New("trees have different number of nodes")
	errTreeMismatchValue = errors.New("tree node values do not match")
)

// walkTree walks a parse tree and sends a string value of each node to the channel.
func walkTree[T any](tr *lexparse.Tree[T], ch chan<- string) {
	defer close(ch)

	doWalkTree(ch, "", tr.Root)
}

// doWalkTree is a recursive worker function used by walkTree.
func doWalkTree[T any](ch chan<- string, depth string, node *lexparse.Node[T]) {
	if node == nil {
		return
	}

	message := ""
	message += fmt.Sprintf(depth+":Value: %v", node.Value)
	ch <- message

	for i, c := range node.Children {
		newDepth := depth + strconv.Itoa(i)
		doWalkTree(ch, newDepth, c)
	}
}

// compareTrees returns true if two trees are equivalent by comparing the
// value of each node in both trees.
func compareTrees[T any](tr1, tr2 *lexparse.Tree[T]) (bool, error) {
	ch1 := make(chan string)
	ch2 := make(chan string)

	go walkTree(tr1, ch1)
	go walkTree(tr2, ch2)

	for {
		i1, more1 := <-ch1
		i2, more2 := <-ch2

		if more1 != more2 {
			return false, errTreeMismatchSize
		}
		if !more1 {
			break
		}

		if i1 != i2 {
			return false, fmt.Errorf("node values: %q, %q, %w", i1, i2, errTreeMismatchValue)
		}
	}

	return true, nil
}

func testExpectedTree(t *testing.T, input string, expected *lexparse.Tree[calcToken]) {

	l := lexparse.NewLexer(runeio.NewReader(strings.NewReader(input)), &lexState{})

	lexemes := l.Lex(context.Background())

	p := lexparse.NewParser[calcToken](lexemes)
	pFn := myParseFn(p)

	ctx := context.Background()
	tree, err := p.Parse(ctx, pFn)
	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}

	// fmt.Printf("\ntree: %+v\n", tree)
	// printTreeNodes(0, tree.Root)
	//
	// fmt.Printf("\nexpected: %+v\n", expected)
	// printTreeNodes(0, expected.Root)

	got, expErr := compareTrees[calcToken](tree, expected)
	if expErr != nil {
		t.Errorf("error expected trees do not match: %s", expErr)
	}
	want := true
	if got != want {
		t.Errorf("trees match: want: %v, got: %v", want, got)
	}
}

func TestAdd(t *testing.T) {
	// t.Parallel()

	input := "1 + 2"

	expected := &lexparse.Tree[calcToken]{
		Root: &lexparse.Node[calcToken]{
			Children: []*lexparse.Node[calcToken]{
				{
					Value: calcToken{
						Type:  addOpToken,
						Value: "+",
					},
					Children: []*lexparse.Node[calcToken]{
						{
							Value: calcToken{
								Type:  natNumberToken,
								Value: "1",
							},
						},
						{
							Value: calcToken{
								Type:  natNumberToken,
								Value: "2",
							},
						},
					},
				},
			},
		},
	}

	testExpectedTree(t, input, expected)
}

func TestAdd2(t *testing.T) {
	// t.Parallel()

	input := "1 + 2 + 3"

	expected := &lexparse.Tree[calcToken]{
		Root: &lexparse.Node[calcToken]{
			Children: []*lexparse.Node[calcToken]{
				{
					Value: calcToken{
						Type:  addOpToken,
						Value: "+",
					},
					Children: []*lexparse.Node[calcToken]{
						{
							Value: calcToken{
								Type:  addOpToken,
								Value: "+",
							},
							Children: []*lexparse.Node[calcToken]{
								{
									Value: calcToken{
										Type:  natNumberToken,
										Value: "1",
									},
								},
								{
									Value: calcToken{
										Type:  natNumberToken,
										Value: "2",
									},
								},
							},
						},
						{
							Value: calcToken{
								Type:  natNumberToken,
								Value: "3",
							},
						},
					},
				},
			},
		},
	}

	testExpectedTree(t, input, expected)
}

func TestAddMul(t *testing.T) {
	// t.Parallel()

	input := "1 + 2 * 3"

	expected := &lexparse.Tree[calcToken]{
		Root: &lexparse.Node[calcToken]{
			Children: []*lexparse.Node[calcToken]{
				{
					Value: calcToken{
						Type:  addOpToken,
						Value: "+",
					},
					Children: []*lexparse.Node[calcToken]{
						{
							Value: calcToken{
								Type:  natNumberToken,
								Value: "1",
							},
						},
						{
							Value: calcToken{
								Type:  mulOpToken,
								Value: "*",
							},
							Children: []*lexparse.Node[calcToken]{
								{
									Value: calcToken{
										Type:  natNumberToken,
										Value: "2",
									},
								},
								{
									Value: calcToken{
										Type:  natNumberToken,
										Value: "3",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	testExpectedTree(t, input, expected)
}

func TestDiv(t *testing.T) {
	// t.Parallel()

	input := "1 / 2"

	expected := &lexparse.Tree[calcToken]{
		Root: &lexparse.Node[calcToken]{
			Children: []*lexparse.Node[calcToken]{
				{
					Value: calcToken{
						Type:  mulOpToken,
						Value: "/",
					},
					Children: []*lexparse.Node[calcToken]{
						{
							Value: calcToken{
								Type:  natNumberToken,
								Value: "1",
							},
						},
						{
							Value: calcToken{
								Type:  natNumberToken,
								Value: "2",
							},
						},
					},
				},
			},
		},
	}

	testExpectedTree(t, input, expected)
}

func TestDiv2(t *testing.T) {
	// t.Parallel()

	input := "1 / 2 / 3"

	expected := &lexparse.Tree[calcToken]{
		Root: &lexparse.Node[calcToken]{
			Children: []*lexparse.Node[calcToken]{
				{
					Value: calcToken{
						Type:  mulOpToken,
						Value: "/",
					},
					Children: []*lexparse.Node[calcToken]{
						{
							Value: calcToken{
								Type:  mulOpToken,
								Value: "/",
							},
							Children: []*lexparse.Node[calcToken]{
								{
									Value: calcToken{
										Type:  natNumberToken,
										Value: "1",
									},
								},
								{
									Value: calcToken{
										Type:  natNumberToken,
										Value: "2",
									},
								},
							},
						},
						{
							Value: calcToken{
								Type:  natNumberToken,
								Value: "3",
							},
						},
					},
				},
			},
		},
	}

	testExpectedTree(t, input, expected)
}

func TestDivMul(t *testing.T) {
	// t.Parallel()

	input := "1 / 2 * 3"

	expected := &lexparse.Tree[calcToken]{
		Root: &lexparse.Node[calcToken]{
			Children: []*lexparse.Node[calcToken]{
				{
					Value: calcToken{
						Type:  mulOpToken,
						Value: "*",
					},
					Children: []*lexparse.Node[calcToken]{
						{
							Value: calcToken{
								Type:  mulOpToken,
								Value: "/",
							},
							Children: []*lexparse.Node[calcToken]{
								{
									Value: calcToken{
										Type:  natNumberToken,
										Value: "1",
									},
								},
								{
									Value: calcToken{
										Type:  natNumberToken,
										Value: "2",
									},
								},
							},
						},
						{
							Value: calcToken{
								Type:  natNumberToken,
								Value: "3",
							},
						},
					},
				},
			},
		},
	}

	testExpectedTree(t, input, expected)
}

func TestMul(t *testing.T) {
	// t.Parallel()

	input := "1 * 2"

	expected := &lexparse.Tree[calcToken]{
		Root: &lexparse.Node[calcToken]{
			Children: []*lexparse.Node[calcToken]{
				{
					Value: calcToken{
						Type:  mulOpToken,
						Value: "*",
					},
					Children: []*lexparse.Node[calcToken]{
						{
							Value: calcToken{
								Type:  natNumberToken,
								Value: "1",
							},
						},
						{
							Value: calcToken{
								Type:  natNumberToken,
								Value: "2",
							},
						},
					},
				},
			},
		},
	}

	testExpectedTree(t, input, expected)
}

func TestMul2(t *testing.T) {
	// t.Parallel()

	input := "1 * 2 * 3"

	expected := &lexparse.Tree[calcToken]{
		Root: &lexparse.Node[calcToken]{
			Children: []*lexparse.Node[calcToken]{
				{
					Value: calcToken{
						Type:  mulOpToken,
						Value: "*",
					},
					Children: []*lexparse.Node[calcToken]{
						{
							Value: calcToken{
								Type:  mulOpToken,
								Value: "*",
							},
							Children: []*lexparse.Node[calcToken]{
								{
									Value: calcToken{
										Type:  natNumberToken,
										Value: "1",
									},
								},
								{
									Value: calcToken{
										Type:  natNumberToken,
										Value: "2",
									},
								},
							},
						},
						{
							Value: calcToken{
								Type:  natNumberToken,
								Value: "3",
							},
						},
					},
				},
			},
		},
	}

	testExpectedTree(t, input, expected)
}

// func TestNumber(t *testing.T) {
// 	// t.Parallel()
//
// 	input := "12"
//
// 	expected := &lexparse.Tree[calcToken]{
// 		Root: &lexparse.Node[calcToken]{
// 			Children: []*lexparse.Node[calcToken]{
// 				{
// 					Value: calcToken{
// 						Type:  natNumberToken,
// 						Value: "1",
// 					},
// 				},
// 			},
// 		},
// 	}
//
// 	testExpectedTree(t, input, expected)
// }

func TestSpace(t *testing.T) {
	// t.Parallel()

	input := "1 +  2"

	expected := &lexparse.Tree[calcToken]{
		Root: &lexparse.Node[calcToken]{
			Children: []*lexparse.Node[calcToken]{
				{
					Value: calcToken{
						Type:  addOpToken,
						Value: "+",
					},
					Children: []*lexparse.Node[calcToken]{
						{
							Value: calcToken{
								Type:  natNumberToken,
								Value: "1",
							},
						},
						{
							Value: calcToken{
								Type:  natNumberToken,
								Value: "2",
							},
						},
					},
				},
			},
		},
	}

	testExpectedTree(t, input, expected)
}

func TestSpaceB(t *testing.T) {
	// t.Parallel()

	input := "1  +  2"

	expected := &lexparse.Tree[calcToken]{
		Root: &lexparse.Node[calcToken]{
			Children: []*lexparse.Node[calcToken]{
				{
					Value: calcToken{
						Type:  addOpToken,
						Value: "+",
					},
					Children: []*lexparse.Node[calcToken]{
						{
							Value: calcToken{
								Type:  natNumberToken,
								Value: "1",
							},
						},
						{
							Value: calcToken{
								Type:  natNumberToken,
								Value: "2",
							},
						},
					},
				},
			},
		},
	}

	testExpectedTree(t, input, expected)
}
