package ego

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// token identifies the type of lex tokens.
type token int

const (
	tokenError        token = iota // error occurred; value is text of error
	tokenEOF                       // end of input
	literals_start                 // start of tokens with meaningful values
	tokenIdentifier                // identifier
	tokenSmallKeyword              // small keyword
	tokenCapKeyword                // capitalized keyword
	tokenArgumentName              // argument name
	tokenOperator                  // operator
	tokenNumber                    // numeric constant
	tokenString                    // string constant
	tokenDelegate                  // identifier '.'
	literals_end                   // end of tokens with meaningful values
	tokenResend                    // 'resend.'
	tokenSelf                      // 'self'
	tokenLeftParen                 // '('
	tokenLeftBracket               // '['
	tokenLeftBrace                 // '{'
	tokenRightParen                // ')'
	tokenRightBracket              // ']'
	tokenRightBrace                // '}'
	tokenBar                       // '|'
	tokenPeriod                    // '.'
	tokenCaret                     // '^'
	tokenLeftArrow                 // '<-'
	tokenEqual                     // '='
	tokenStar                      // '*'
)

var tokens = [...]string{
	tokenError:        "error",
	tokenEOF:          "EOF",
	tokenIdentifier:   "identifier",
	tokenSmallKeyword: "small-keyword",
	tokenCapKeyword:   "capitalized-keyword",
	tokenArgumentName: "argument-name",
	tokenOperator:     "operator",
	tokenNumber:       "number",
	tokenString:       "string",
	tokenDelegate:     "delegate",
	tokenResend:       "resend",
	tokenSelf:         "self",
	tokenLeftParen:    "(",
	tokenLeftBracket:  "[",
	tokenLeftBrace:    "{",
	tokenRightParen:   ")",
	tokenRightBracket: "]",
	tokenRightBrace:   "}",
	tokenBar:          "|",
	tokenPeriod:       ".",
	tokenCaret:        "^",
	tokenLeftArrow:    "<-",
	tokenEqual:        "=",
	tokenStar:         "*",
}

func (t token) isLiteral() bool { return literals_start < t && t < literals_end }

const eof = 0

// item represents a token returned from the scanner.
type item struct {
	t   token  // Type, such as tokenNumber.
	v   string // Value, such as "23.2".
	pos int
}

func (i item) String() string {
	switch i.t {
	case tokenEOF:
		return "EOF"
	case tokenError:
		return i.v
	}
	if len(i.v) > 10 {
		return fmt.Sprintf("%.10q...", i.v)
	}
	return fmt.Sprintf("%q", i.v)
}

// lexer holds the state of the scanner.
type lexer struct {
	name  string      // Used only for error reports.
	input string      // The string being scanned.
	start int         // Start position of this item.
	pos   int         // Current position in the input.
	width int         // Width of last rune read from input.
	items chan<- item // Channel of scanned items.
}

type stateFn func(*lexer) stateFn

func (l *lexer) emit(t token) {
	l.items <- item{t, l.input[l.start:l.pos], l.start}
	l.start = l.pos
}

// next returns the next rune in the input.
func (l *lexer) next() (r rune) {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
}

// backup steps back one rune.
// Can be called only once per call of next.
func (l *lexer) backup() {
	l.pos -= l.width
	l.width = 0
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// accept consumes the next rune if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.ContainsRune(valid, l.next()) {
		return true
	}
	l.backup()
	return false
}

// acceptRun consumes a run of runes from the valid set.
func (l *lexer) acceptRun(valid string) {
	for strings.ContainsRune(valid, l.next()) {
	}
	l.backup()
}

// errorf returns an error token and terminates the scan by passing back a nil
// pointer that will be the next state, terminating l.run.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{tokenError, fmt.Sprintf(format, args...), l.start}
	return nil
}

// // nextItem returns the next item from the input.
// func (l *lexer) nextItem() item {
// 	for {
// 		select {
// 		case item := <-l.items:
// 			return item
// 		default:
// 			l.state = l.state(l)
// 		}
// 	}
// 	panic("unreachable")
// }

func lex(name, input string) <-chan item {
	items := make(chan item)

	go func() {
		l := &lexer{
			name:  name,
			input: input,
			items: items,
		}
		for state := lexTop; state != nil; state = state(l) {
		}
		close(l.items) // No more tokens will be delivered.
	}()

	return items
}

const (
	operatorChars   = "!@#$%^&*-+=~/?<>,;|‘\\"
	smallLetter     = "abcdefghijklmnopqrstuvwxyz"
	capitalLetter   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digit           = "0123456789"
	identifierStart = smallLetter + digit + "_"
	identifierChars = identifierStart + capitalLetter
	resendSucc      = identifierStart + operatorChars
)

func lexTop(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case r == eof:
			l.emit(tokenEOF)
			return nil
		case unicode.IsSpace(r):
			l.ignore()
		case strings.ContainsRune(operatorChars, r):
			return lexOperator
		case r == '.':
			l.emit(tokenPeriod)
		case strings.ContainsRune(identifierStart, r):
			return lexIdentifier
		case 'A' <= r && r <= 'Z':
			return lexCapKeyword
		case r == ':':
			return lexArgumentName
		case r == '-' || '0' <= r && r <= '9':
			l.backup()
			return lexNumber
		case r == '\'':
			return lexString
		case r == '"':
			return lexComment
		case r == '(':
			l.emit(tokenLeftParen)
		case r == '{':
			l.emit(tokenLeftBrace)
		case r == '[':
			l.emit(tokenLeftBracket)
		case r == ')':
			l.emit(tokenRightParen)
		case r == '}':
			l.emit(tokenRightBrace)
		case r == ']':
			l.emit(tokenRightBracket)
		}
	}
}

func lexOperator(l *lexer) stateFn {
	l.acceptRun(operatorChars)
	switch l.input[l.start:l.pos] {
	case "<-":
		l.emit(tokenLeftArrow)
	case "=":
		l.emit(tokenEqual)
	case "|":
		l.emit(tokenBar)
	case "^":
		l.emit(tokenCaret)
	case "\\":
		if c := l.peek(); c != '\n' && c != '\r' {
			l.emit(tokenOperator)
		}
	default:
		l.emit(tokenOperator)
	}
	return lexTop
}

func (l *lexer) resend() stateFn {
	switch l.input[l.start:l.pos] {
	case "resend.":
		l.emit(tokenResend)
	case "self.":
		return l.errorf("using 'self' as a parent name for a directed resend")
	default:
		l.emit(tokenDelegate)
	}
	return lexTop
}

func (l *lexer) identifier() stateFn {
	switch l.input[l.start:l.pos] {
	case "self":
		l.emit(tokenSelf)
	case "resend":
		return l.errorf("using 'resend' outside of a resend")
	default:
		l.emit(tokenIdentifier)
	}
	return lexTop
}

func (l *lexer) argumentName() stateFn {
	switch l.input[l.start:l.pos] {
	case ":self":
		return l.errorf("using 'self' as an argument")
	case ":resend":
		return l.errorf("using 'resend' as an argument")
	}
	l.emit(tokenArgumentName)
	return lexTop
}

func lexIdentifier(l *lexer) stateFn {
	l.acceptRun(identifierChars)
	switch {
	case l.accept(":"):
		l.emit(tokenSmallKeyword)
		return lexTop
	case l.accept("."):
		w := l.width
		if strings.ContainsRune(resendSucc, l.peek()) {
			return l.resend()
		}
		l.pos -= w
	}
	return l.identifier()
}

func lexCapKeyword(l *lexer) stateFn {
	l.acceptRun(identifierChars)
	if l.accept(":") {
		l.emit(tokenCapKeyword)
		return lexTop
	}
	return l.errorf("expected ':', found %q", l.peek())
}

func lexArgumentName(l *lexer) stateFn {
	if l.accept(identifierStart) {
		l.acceptRun(identifierChars)
		return l.argumentName()
	}
	return l.errorf("expected lowercase letter or '_', found %q", l.peek())
}

func lexComment(l *lexer) stateFn {
	r := l.next()
	for r != '"' && r != eof {
		r = l.next()
	}
	if r == '"' {
		l.ignore()
		return lexTop
	}
	return l.errorf("unclosed comment")
}

func lexString(l *lexer) stateFn {
	// TODO
	return nil
}

func lexNumber(l *lexer) stateFn {
	// TODO
	return nil
}

// number  → [ ‘-’ ] (integer | real)
// integer → [base] general-digit {general-digit}
// real  → fixed-point | float
// fixed-point → decimal ‘.’ decimal
// float → decimal [ ‘.’ decimal ] (‘e’ | ‘E’) [ ‘+’ | ‘-’ ] decimal
// general-digit → digit | letter
// decimal → digit {digit}
// base  → decimal (‘r’ | ‘R’)
// string  → ‘’’ { normal-char | escape-char } ‘’’
// normal-char → any character except ‘\’ and ‘’’
// escape-char → ‘\t’ | ‘\b’ | ‘\n’ | ‘\f’ | ‘\r’ | ‘\v’ | ‘\a’ | ‘\0’ | ‘\\’ | ‘\’’ | ‘\”’ | ‘\?’ | numeric-escape
// numeric-escape  → ‘\x’ general-digit general-digit | ( ‘\d’ | ‘\o’ ) digit digit digit
