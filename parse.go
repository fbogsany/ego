package ego

type parser struct {
	item
	peekItem, nextItem <-chan item
	pushBack           chan<- item
	quit               chan<- struct{}
}

func parse(name, input string) *parser {
	peek, next, push := make(chan item), make(chan item), make(chan item)
	quit := make(chan struct{})
	defer close(quit)

	go func() {
		items := lex(name, input)
		i := <-items
		backup, hasBackup := i, false
		for {
			select {
			case peek <- i:
			case next <- i:
				if hasBackup {
					i, hasBackup = backup, false
				} else {
					i = <-items
				}
			case item := <-push:
				backup, hasBackup = i, true
				i = item
			case <-quit:
				return
			}
		}
	}()

	p := &parser{peekItem: peek, nextItem: next, pushBack: push, quit: quit}
	p.next()
	return p
}

func (p *parser) next()       { p.item = <-p.nextItem }
func (p *parser) peek() item  { return <-p.peekItem }
func (p *parser) atEOF() bool { return p.peek().t == tokenEOF }

func (p *parser) expect(t token) int {
	pos := p.pos
	if p.t != t {
		p.errorExpected(pos, "'"+tokens[t]+"'")
	}
	p.next()
	return pos
}

func (p *parser) error(pos int, msg string) {
	// TODO
	panic(msg)
}

func (p *parser) errorExpected(pos int, msg string) {
	msg = "expected " + msg
	if pos == p.pos {
		msg += ", found '" + tokens[p.t] + "'"
		if p.t.isLiteral() {
			msg += " " + p.v
		}
	}
	p.error(pos, msg)
}

var implicitSelf expr = nil

func (p *parser) parseExpr() expr {
	if p.next(); p.t == tokenEOF {
		return nil
	}
	return p.parsePrimaryExpr()
}

func isKeyword(t token) bool { return t == tokenSmallKeyword }

func isOperator(t token) bool {
	return t == tokenOperator || t == tokenEqual || t == tokenLeftArrow // TODO || t == tokenTilde?
}

func (p *parser) maybeOperator() bool {
	i := p.peek()
	return isOperator(i.t) || i.t == tokenNumber && i.v[0] == '-'
}

func (p *parser) parsePrimaryExpr() (e expr) {
	d := p.parseDelegate(isKeyword)
	if p.t == tokenSmallKeyword {
		e = implicitSelf
	} else {
		if e = p.parseBinary(); e == nil {
			return nil
		}
		if p.next(); p.t == tokenEOF {
			// TODO error
			return nil
		}
	}
	if p.t == tokenSmallKeyword {
		kw := []string{p.v}
		arg := p.parseExpr()
		if arg == nil {
			return nil
		}
		args := []expr{arg}
		for p.next(); p.t == tokenCapKeyword; p.next() {
			kw = append(kw, p.v)
			if arg = p.parseExpr(); arg == nil {
				return nil
			}
			args = append(args, arg)
		}
		if p.t == tokenEOF {
			// TODO error
			return nil
		}
		e = &keyword{e, kw, args, d}
	}
	return
}

func (p *parser) parseDelegate(expectNext func(token) bool) string {
	if p.t == tokenDelegate || p.t == tokenResend {
		if expectNext(p.peek().t) {
			d := p.v[:len(p.v)-1]
			p.next()
			return d
		}
	}
	return ""
}

func (p *parser) parseBinary() (e expr) {
	d := p.parseDelegate(isOperator)
	if isOperator(p.t) {
		e = implicitSelf
	} else {
		if p.t == tokenLeftParen {
			if p.atEOF() {
				// TODO error
				return nil
			}
		}
		if e = p.parseUnary(); e == nil {
			return nil
		}
		if p.atEOF() {
			// TODO error
			return nil
		}
		// TODO method candidate?
	}
	prev := ""
	for p.maybeOperator() {
		op := p.v
		if !isOperator(p.t) {
			op = op[:1]
		}
		if len(prev) != 0 && prev != op {
			// TODO syntax error
			return nil
		}
		prev = op
		var arg expr
		switch p.next(); p.t {
		case tokenEOF:
			// TODO error
			return nil
		case tokenSmallKeyword:
			arg = p.parseExpr()
		default:
			arg = p.parseUnary()
		}
		if arg == nil {
			return nil
		}
		e = &binary{e, op, arg, d}
		if p.atEOF() {
			// TODO error
			return nil
		}
	}
	return
}

func (p *parser) parseUnary() (e expr) {
	// TODO
	return
}
