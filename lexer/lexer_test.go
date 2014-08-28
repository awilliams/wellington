package lexer_test

import (
	"fmt"
	"testing"

	"github.com/drewwells/sprite-sass/lexer"
)

// This example shows a trivial parser using Advance, the lowest level lexer
// function.  The parser decodes a serialization format for test status
// messages generated by a hypothetical test suite. The rune '.' is translated
// using the format "%d success", the rune '!' is translated using "%d
// failure".
/*func ExampleLexer_advance() {

	// delare token types as constants
	const (
		itemOK lexer.ItemType = iota
		itemFail
	)

	// create a StateFn to parse the language.
	var start lexer.StateFn
	start = func(lex *lexer.Lexer) lexer.StateFn {
		return lex.Action()
		c, n := lex.Advance()
		if lexer.IsEOF(c, n) {
			return nil
		}
		if lexer.IsInvalid(c, n) {
			return lex.Errorf("invalid utf-8 rune")
		}
		switch c {
		case '.':
			lex.Emit(itemOK)
		case '!':
			lex.Emit(itemFail)
		default:
			// lex.Backup() does not need to be called even though lex.Pos()
			// points at the next rune. The position of the error is the start
			// of the current lexeme (in this case the unexpected rune we just
			// read).
			return lex.Errorf("unexpected rune %q", c)
		}
		return start
	}

	// create a parser for the language.
	parse := func(input string) ([]string, error) {
		lex := lexer.New(start, input)
		var status []string
		for {
			item := lex.Next()
			err := item.Err()
			if err != nil {
				return nil, fmt.Errorf("%v (pos %d)", err, item.Pos)
			}
			switch item.Type {
			case lexer.ItemEOF:
				return status, nil
			case itemOK:
				status = append(status, fmt.Sprintf("%d success", item.Pos))
			case itemFail:
				status = append(status, fmt.Sprintf("%d failure", item.Pos))

			default:
				panic(fmt.Sprintf("unexpected item %0x (pos %d)", item.Type, item.Pos))
			}
		}
	}

	// parse a valid string and print the status
	status, err := parse(".!")
	fmt.Printf("%q %v\n", status, err)

	// parse an invalid string and print the error
	status, err = parse("!.!?.")
	fmt.Printf("%q %v\n", status, err)

	// Output:
	// ["0 success" "1 failure"] <nil>
	// [] unexpected rune '?' (pos 3)
}*/

func TestSassLexer(t *testing.T) {

	// delare token types as constants
	const (
		itemOK lexer.ItemType = iota
		itemFail
	)

	// create a StateFn to parse the language.
	var start lexer.StateFn
	start = func(lex *lexer.Lexer) lexer.StateFn {
		return lex.Action()
	}

	// create a parser for the language.
	parse := func(input string) ([]string, error) {
		lex := lexer.New(start, input)

		var status []string
		for {
			item := lex.Next()
			err := item.Err()
			//fmt.Printf("Item: %s\n", item)
			if err != nil {
				return nil, fmt.Errorf("Error: %v (pos %d)", err, item.Pos)
			}
			switch item.Type {
			case lexer.ItemEOF:
				return status, nil
			case lexer.SPRITE, lexer.TEXT:
				status = append(status, fmt.Sprintf("%s", item))
			default:
				fmt.Printf("Extra %s %d\n", item.Type, item.Pos)
			}
		}
	}

	// parse a valid string and print the status
	status, err := parse(`sprite($images,"one.png");`)

	fmt.Printf("Status: %q %v\n", status, err)

	// Output:
	// ["0 success" "1 failure"] <nil>
	// [] unexpected rune '?' (pos 3)
}
