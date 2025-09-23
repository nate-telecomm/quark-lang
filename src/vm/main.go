package vm

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

/* ---------- Opcodes ---------- */
type Op byte

const (
	OpHalt Op = iota
	OpLoadConst      // operand: u16 const index
	OpStoreLocal     // operand: u16 local index
	OpLoadLocal      // operand: u16 local index
	OpAdd            // pop a,b push a+b
	OpSub
	OpMul
	OpDiv
	OpCallBuiltin    // operand: u8 builtin id, operand: u8 argc
	OpPop
	OpJump           // operand: u16 addr
	OpJumpIfFalse    // operand: u16 addr
)

/* ---------- Lexer ---------- */
type TokenKind int

const (
	TokEOF TokenKind = iota
	TokIdent
	TokNumber
	TokString
	TokLet
	TokPrint
	TokAssign   // =
	TokSemi     // ;
	TokLParen   // (
	TokRParen   // )
	TokPlus
	TokMinus
	TokStar
	TokSlash
	TokUnknown
)

type Token struct {
	Kind  TokenKind
	Value string
	Pos   int
}

type Lexer struct {
	input string
	pos   int
}

func NewLexer(s string) *Lexer { return &Lexer{input: s} }

func (l *Lexer) next() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	r := rune(l.input[l.pos])
	l.pos++
	return r
}
func (l *Lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return rune(l.input[l.pos])
}

func (l *Lexer) skipSpace() {
	for unicode.IsSpace(l.peek()) {
		l.next()
	}
}

func (l *Lexer) readWhile(pred func(rune) bool) string {
	var b strings.Builder
	for pred(l.peek()) && l.peek() != 0 {
		b.WriteRune(l.next())
	}
	return b.String()
}

func (l *Lexer) NextToken() Token {
	l.skipSpace()
	start := l.pos
	ch := l.peek()
	if ch == 0 {
		return Token{Kind: TokEOF, Pos: start}
	}
	if unicode.IsLetter(ch) || ch == '_' {
		s := l.readWhile(func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' })
		switch s {
		case "let":
			return Token{Kind: TokLet, Value: s, Pos: start}
		case "print":
			return Token{Kind: TokPrint, Value: s, Pos: start}
		default:
			return Token{Kind: TokIdent, Value: s, Pos: start}
		}
	}
	// numbers
	if unicode.IsDigit(ch) {
		num := l.readWhile(func(r rune) bool { return unicode.IsDigit(r) })
		return Token{Kind: TokNumber, Value: num, Pos: start}
	}
	// strings : "..."
	if ch == '"' {
		l.next()
		var b strings.Builder
		for {
			c := l.next()
			if c == 0 {
				return Token{Kind: TokUnknown, Pos: start}
			}
			if c == '"' {
				break
			}
			if c == '\\' {
				n := l.next()
				if n == 'n' {
					b.WriteByte('\n')
				} else {
					b.WriteRune(n)
				}
			} else {
				b.WriteRune(c)
			}
		}
		return Token{Kind: TokString, Value: b.String(), Pos: start}
	}
	switch l.next() {
	case '=':
		return Token{Kind: TokAssign, Pos: start}
	case ';':
		return Token{Kind: TokSemi, Pos: start}
	case '(':
		return Token{Kind: TokLParen, Pos: start}
	case ')':
		return Token{Kind: TokRParen, Pos: start}
	case '+':
		return Token{Kind: TokPlus, Pos: start}
	case '-':
		return Token{Kind: TokMinus, Pos: start}
	case '*':
		return Token{Kind: TokStar, Pos: start}
	case '/':
		return Token{Kind: TokSlash, Pos: start}
	default:
		return Token{Kind: TokUnknown, Pos: start}
	}
}

/* ---------- Parser ---------- */

type Expr interface{}
type NumberLiteral struct{ Val int64 }
type StringLiteral struct{ Val string }
type Ident struct{ Name string }
type Binary struct {
	Op    TokenKind
	Left  Expr
	Right Expr
}
type Call struct {
	Callee string
	Args   []Expr
}

type Stmt interface{}
type LetStmt struct {
	Name string
	Val  Expr
}
type ExprStmt struct {
	E Expr
}

type Parser struct {
	lex  *Lexer
	cur  Token
	peek Token
}

func NewParser(src string) *Parser {
	l := NewLexer(src)
	p := &Parser{lex: l}
	p.cur = p.lex.NextToken()
	p.peek = p.lex.NextToken()
	return p
}

func (p *Parser) advance() {
	p.cur = p.peek
	p.peek = p.lex.NextToken()
}

func (p *Parser) expect(kind TokenKind) error {
	if p.cur.Kind == kind {
		return nil
	}
	return fmt.Errorf("expected token %v, got %v", kind, p.cur.Kind)
}

func (p *Parser) parseProgram() ([]Stmt, error) {
	var out []Stmt
	for p.cur.Kind != TokEOF {
		st, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, nil
}

func (p *Parser) parseStatement() (Stmt, error) {
	if p.cur.Kind == TokLet {
		p.advance()
		if p.cur.Kind != TokIdent {
			return nil, fmt.Errorf("expected identifier after let")
		}
		name := p.cur.Value
		p.advance()
		if p.cur.Kind != TokAssign {
			return nil, fmt.Errorf("expected = after identifier")
		}
		p.advance()
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if p.cur.Kind == TokSemi {
			p.advance()
		}
		return LetStmt{Name: name, Val: expr}, nil
	}
	// expression statement
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.cur.Kind == TokSemi {
		p.advance()
	}
	return ExprStmt{E: expr}, nil
}

func (p *Parser) parseExpression() (Expr, error) {
	return p.parseBinary(0)
}

var precedence = map[TokenKind]int{
	TokPlus:  10,
	TokMinus: 10,
	TokStar:  20,
	TokSlash: 20,
}

func (p *Parser) parseBinary(minPrec int) (Expr, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		op := p.cur.Kind
		prec, ok := precedence[op]
		if !ok || prec < minPrec {
			break
		}
		p.advance()
		right, err := p.parseBinary(prec + 1)
		if err != nil {
			return nil, err
		}
		left = Binary{Op: op, Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parsePrimary() (Expr, error) {
	switch p.cur.Kind {
	case TokNumber:
		n, _ := strconv.ParseInt(p.cur.Value, 10, 64)
		v := NumberLiteral{Val: n}
		p.advance()
		return v, nil
	case TokString:
		v := StringLiteral{Val: p.cur.Value}
		p.advance()
		return v, nil
	case TokIdent, TokPrint:
		name := p.cur.Value
		p.advance()
		if p.cur.Kind == TokLParen {
			// call
			p.advance()
			var args []Expr
			if p.cur.Kind != TokRParen {
				for {
					arg, err := p.parseExpression()
					if err != nil {
						return nil, err
					}
					args = append(args, arg)
					if p.cur.Kind == TokRParen {
						break
					}
					break
				}
			}
			if p.cur.Kind == TokRParen {
				p.advance()
			}
			return Call{Callee: name, Args: args}, nil
		}
		return Ident{Name: name}, nil
	case TokLParen:
		p.advance()
		e, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if p.cur.Kind == TokRParen {
			p.advance()
		}
		return e, nil
	default:
		return nil, fmt.Errorf("unexpected token in primary: %v", p.cur)
	}
}

/* ---------- Compiler (AST -> bytecode + constants) ---------- */

type Compiler struct {
	consts   []interface{}
	code     []byte
	locals   map[string]uint16 // map local name -> index
	nextLoc  uint16
}

func NewCompiler() *Compiler {
	return &Compiler{
		consts:  []interface{}{},
		code:    []byte{},
		locals:  map[string]uint16{},
		nextLoc: 0,
	}
}
func (c *Compiler) addConst(v interface{}) uint16 {
	for i, x := range c.consts {
		if x == v {
			return uint16(i)
		}
	}
	c.consts = append(c.consts, v)
	return uint16(len(c.consts) - 1)
}
func (c *Compiler) emit(b ...byte) {
	c.code = append(c.code, b...)
}
func (c *Compiler) emitU16(u uint16) {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, u)
	c.emit(b...)
}

func (c *Compiler) compileExpr(e Expr) error {
	switch v := e.(type) {
	case NumberLiteral:
		idx := c.addConst(v.Val)
		c.emit(byte(OpLoadConst))
		c.emitU16(idx)
	case StringLiteral:
		idx := c.addConst(v.Val)
		c.emit(byte(OpLoadConst))
		c.emitU16(idx)
	case Ident:
		li, ok := c.locals[v.Name]
		if !ok {
			return fmt.Errorf("unknown identifier %s", v.Name)
		}
		c.emit(byte(OpLoadLocal))
		c.emitU16(li)
	case Binary:
		if err := c.compileExpr(v.Left); err != nil {
			return err
		}
		if err := c.compileExpr(v.Right); err != nil {
			return err
		}
		switch v.Op {
		case TokPlus:
			c.emit(byte(OpAdd))
		case TokMinus:
			c.emit(byte(OpSub))
		case TokStar:
			c.emit(byte(OpMul))
		case TokSlash:
			c.emit(byte(OpDiv))
		default:
			return fmt.Errorf("unknown binary op")
		}
	case Call:
		for _, a := range v.Args {
			if err := c.compileExpr(a); err != nil {
				return err
			}
		}
		if v.Callee == "print" {
			argc := byte(len(v.Args))
			c.emit(byte(OpCallBuiltin))
			c.emit(argc)
		} else {
			return fmt.Errorf("unknown function %s", v.Callee)
		}
	default:
		return fmt.Errorf("unknown expr type %T", v)
	}
	return nil
}

func (c *Compiler) compileStmt(s Stmt) error {
	switch st := s.(type) {
	case LetStmt:
		idx := c.nextLoc
		c.nextLoc++
		c.locals[st.Name] = idx
		if err := c.compileExpr(st.Val); err != nil {
			return err
		}
		c.emit(byte(OpStoreLocal))
		c.emitU16(idx)
	case ExprStmt:
		if err := c.compileExpr(st.E); err != nil {
			return err
		}
		c.emit(byte(OpPop))
	default:
		return fmt.Errorf("unknown stmt type %T", st)
	}
	return nil
}

func (c *Compiler) compileProgram(stmts []Stmt) ([]byte, []interface{}, error) {
	for _, s := range stmts {
		if err := c.compileStmt(s); err != nil {
			return nil, nil, err
		}
	}
	c.emit(byte(OpHalt))
	return c.code, c.consts, nil
}

/* ---------- Serializer / Deserializer ---------- */

// Format: [4-byte len][const JSON bytes][bytecode bytes]
func SerializeBytecode(code []byte, consts []interface{}) ([]byte, error) {
	j, err := json.Marshal(consts)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(nil)
	ln := uint32(len(j))
	binary.Write(buf, binary.LittleEndian, ln)
	buf.Write(j)
	buf.Write(code)
	return buf.Bytes(), nil
}

func DeserializeBytecode(blob []byte) ([]byte, []interface{}, error) {
	if len(blob) < 4 {
		return nil, nil, errors.New("blob too small")
	}
	ln := binary.LittleEndian.Uint32(blob[:4])
	if int(4+ln) > len(blob) {
		return nil, nil, errors.New("invalid length")
	}
	constJSON := blob[4 : 4+ln]
	var consts []interface{}
	if err := json.Unmarshal(constJSON, &consts); err != nil {
		return nil, nil, err
	}
	code := blob[4+ln:]
	return code, consts, nil
}

/* ---------- VM ---------- */

type Value interface{}

func RunBytecode(blob []byte) error {
	code, consts, err := DeserializeBytecode(blob)
	if err != nil {
		return err
	}
	ip := 0
	var stack []Value
	locals := map[uint16]Value{}

	push := func(v Value) { stack = append(stack, v) }
	pop := func() (Value, error) {
		if len(stack) == 0 {
			return nil, errors.New("stack empty")
		}
		v := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		return v, nil
	}

	readU16 := func() (uint16, error) {
		if ip+2 > len(code) {
			return 0, errors.New("read past end")
		}
		x := binary.LittleEndian.Uint16(code[ip : ip+2])
		ip += 2
		return x, nil
	}

	for {
		if ip >= len(code) {
			return errors.New("ip out of range")
		}
		op := Op(code[ip])
		ip++
		switch op {
		case OpHalt:
			return nil
		case OpLoadConst:
			idx, err := readU16()
			if err != nil {
				return err
			}
			if int(idx) >= len(consts) {
				return fmt.Errorf("const idx out of range")
			}
			push(consts[idx])
		case OpStoreLocal:
			li, err := readU16()
			if err != nil {
				return err
			}
			v, err := pop()
			if err != nil {
				return err
			}
			locals[li] = v
		case OpLoadLocal:
			li, err := readU16()
			if err != nil {
				return err
			}
			v, ok := locals[li]
			if !ok {
				return fmt.Errorf("uninitialized local %d", li)
			}
			push(v)
		case OpAdd, OpSub, OpMul, OpDiv:
			bv, err := pop()
			if err != nil {
				return err
			}
			av, err := pop()
			if err != nil {
				return err
			}
			switch a := av.(type) {
			case float64:
				af := a
				bf := 0.0
				switch bb := bv.(type) {
				case float64:
					bf = bb
				case string:
					bf, _ = strconv.ParseFloat(bb, 64)
				}
				var res float64
				if op == OpAdd {
					res = af + bf
				} else if op == OpSub {
					res = af - bf
				} else if op == OpMul {
					res = af * bf
				} else {
					res = af / bf
				}
				push(res)
			case string:
				if op == OpAdd {
					if bs, ok := bv.(string); ok {
						push(a + bs)
						continue
					}
				}
				return fmt.Errorf("unsupported operand types for op")
			default:
				return fmt.Errorf("unsupported operand a type %T", a)
			}
		case OpCallBuiltin:
			if ip >= len(code) {
				return fmt.Errorf("unexpected eof for builtin")
			}
			argc := int(code[ip])
			ip++
			if len(stack) < argc {
				return fmt.Errorf("stack underflow for call, want %d", argc)
			}

			args := make([]Value, argc)
			for i := argc - 1; i >= 0; i-- {
				v := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				args[i] = v
			}

			out := make([]string, len(args))
			for i, a := range args {
				switch x := a.(type) {
				case float64:
					out[i] = strconv.FormatFloat(x, 'f', -1, 64)
				case string:
					out[i] = x
				default:
					out[i] = fmt.Sprintf("%v", x)
				}
			}
			fmt.Println(strings.Join(out, " "))

			push(nil)
		case OpPop:
			if _, err := pop(); err != nil {
				return err
			}
		case OpJump:
			addr, err := readU16()
			if err != nil {
				return err
			}
			ip = int(addr)
		case OpJumpIfFalse:
			addr, err := readU16()
			if err != nil {
				return err
			}
			v, err := pop()
			if err != nil {
				return err
			}
			sf := false
			switch x := v.(type) {
			case float64:
				sf = x == 0
			case string:
				sf = x == ""
			default:
				sf = false
			}
			if sf {
				ip = int(addr)
			}
		default:
			return fmt.Errorf("unknown opcode %d", op)
		}
	}
}

/* ---------- Glue: compile source -> blob ---------- */

func CompileSourceToBlob(src string) ([]byte, error) {
	p := NewParser(src)
	stmts, err := p.parseProgram()
	if err != nil {
		return nil, err
	}
	c := NewCompiler()
	code, consts, err := c.compileProgram(stmts)
	if err != nil {
		return nil, err
	}
	blob, err := SerializeBytecode(code, consts)
	if err != nil {
		return nil, err
	}
	return blob, nil
}
