package ego

import (
	"testing"
)

type test struct {
	source string
	tokens []token
}

func TestLex(t *testing.T) {
	tests := []test{
		{"", []token{}},
		{"  <- :arg", []token{tokenLeftArrow, tokenArgumentName}},
	}
	for i, test := range tests {
		test.test(t, i)
	}
}

func (test *test) test(t *testing.T, n int) {
	items := lex("test", test.source)
	for i, expected := range test.tokens {
		if item := <-items; item.t != expected {
			t.Errorf("[%d] expected %s but found %s (%s) at %d", n, tokens[expected], tokens[item.t], item, i)
		}
	}
	if item := <-items; item.t != tokenEOF {
		t.Errorf("[%d] expected EOF but found %s (%s)", n, tokens[item.t], item)
	}
}
