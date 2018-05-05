package vm

import (
	"github.com/goby-lang/goby/compiler"
	"github.com/goby-lang/goby/compiler/bytecode"
	"github.com/goby-lang/goby/compiler/lexer"
	"github.com/goby-lang/goby/compiler/parser"
	"github.com/goby-lang/goby/compiler/token"
	"github.com/goby-lang/goby/vm/classes"
	"github.com/goby-lang/goby/vm/errors"
	"strings"
)

// Ripper is a loadable library and has abilities to obtain parsed/lexed/tokenized Goby codes from String.
// The library would be convenient for validating Goby codes when building lint tools,
// as well as the tests for Goby's compiler.
// For now, Ripper is a class and has only class methods, but I think this should finally be a 'newable' module with instance methods.

// Class methods --------------------------------------------------------
func builtInRipperClassMethods() []*BuiltinMethodObject {
	return []*BuiltinMethodObject{
		{
			Name: "new",
			Fn: func(receiver Object, sourceLine int) builtinMethodBody {
				return func(t *thread, args []Object, blockFrame *normalCallFrame) Object {
					return t.vm.initUnsupportedMethodError(sourceLine, "#new", receiver)
				}
			},
		},
		{
			// Returns the list of instruction code generated from Goby code.
			// Returns an error when the code is invalid.
			// The return value is a "tuple" style:
			// - `Array`: contains an instruction set
			//   - `arg_types:` (none if `nil`)
			// 		 - `names:` array of names (string)
			//     - `types:` array of types (integer)
			//   - `instructions:` array of instructions
			//     - `action:` string
			//     - `anchor:` integer
			//     - `line:` integer
			//     - `params:` array of parameters (string)
			//     - `source_line:` integer
			//     - `arg_set:` (none if `nil`)
			//   		 - `names:` array of names (string)
			//       - `types:` array of types (integer)
			//
			// ```ruby
			// require 'ripper'; Ripper.instruction "10.times do |i| puts i end"
			// #=>
			//
			// require 'ripper'; Ripper.instruction "10.times do |i| puts i" # the code is invalid
			// #=> TypeError: InternalError%!(EXTRA string=String, string=Invalid Goby code)
			// ```
			//
			// @param Goby code [String]
			// @return [String]
			Name: "instruction",
			Fn: func(receiver Object, sourceLine int) builtinMethodBody {
				return func(t *thread, args []Object, blockFrame *normalCallFrame) Object {
					if len(args) != 1 {
						return t.vm.initErrorObject(errors.ArgumentError, sourceLine, "Expect 1 argument. got=%d", len(args))
					}

					arg := args[0]
					switch arg.(type) {
					case *StringObject:
					default:
						return t.vm.initErrorObject(errors.TypeError, sourceLine, errors.WrongArgumentTypeFormat, classes.StringClass, arg.Class().Name)
					}

					i, err := compiler.CompileToInstructions(arg.toString(), NormalMode)
					if err != nil {
						return t.vm.initErrorObject(errors.TypeError, sourceLine, errors.InternalError, classes.StringClass, errors.InvalidGobyCode)
					}

					return t.vm.convertToTuple(i)
				}
			},
		},
		{
			// Returns a nested array that contains the line #, type of the token, and the literal of the token.
			// Note that the class method does not return any errors even though the provided Goby code is invalid.
			//
			// ```ruby
			// require 'ripper'; Ripper.lex "10.times do |i| puts i end"
			// #=> [[0, "on_int", "10"], [0, "on_dot", "."], [0, "on_ident", "times"], [0, "on_do", "do"], [0, "on_bar", "|"], [0, "on_ident", "i"], [0, "on_bar", "|"], [0, "on_ident", "puts"], [0, "on_ident", "i"], [0, "on_end", "end"], [0, "on_eof", ""]]
			//
			// require 'ripper'; Ripper.lex "10.times do |i| puts i" # the code is invalid
			// #=> [[0, "on_int", "10"], [0, "on_dot", "."], [0, "on_ident", "times"], [0, "on_do", "do"], [0, "on_bar", "|"], [0, "on_ident", "i"], [0, "on_bar", "|"], [0, "on_ident", "puts"], [0, "on_ident", "i"], [0, "on_eof", ""]]
			// ```
			//
			// @param Goby code [String]
			// @return [Array]
			Name: "lex",
			Fn: func(receiver Object, sourceLine int) builtinMethodBody {
				return func(t *thread, args []Object, blockFrame *normalCallFrame) Object {
					if len(args) != 1 {
						return t.vm.initErrorObject(errors.ArgumentError, sourceLine, "Expect 1 argument. got=%d", len(args))
					}

					arg := args[0]
					switch arg.(type) {
					case *StringObject:
					default:
						return t.vm.initErrorObject(errors.TypeError, sourceLine, errors.WrongArgumentTypeFormat, classes.StringClass, arg.Class().Name)
					}

					l := lexer.New(arg.toString())
					el := t.vm.initArrayObject([]Object{})
					eli := []Object{}
					var nt token.Token
					for i := 0; ; i++ {
						nt = l.NextToken()
						eli = append(eli, t.vm.initIntegerObject(nt.Line))
						eli = append(eli, t.vm.initStringObject(convertLex(nt.Type)))
						eli = append(eli, t.vm.initStringObject(nt.Literal))
						el.Elements = append(el.Elements, t.vm.initArrayObject(eli))
						if nt.Type == token.EOF {
							break
						}
						eli = nil
					}
					return el
				}
			},
		},
		{
			// Returns the parsed Goby codes as a String object.
			// Returns an error when the code is invalid.
			//
			// ```ruby
			// require 'ripper'; Ripper.parse "10.times do |i| puts i end"
			// #=> "10.times() do |i|
			// #=> self.puts(i)
			// #=> end"
			//
			// require 'ripper'; Ripper.parse "10.times do |i| puts i" # the code is invalid
			// #=> TypeError: InternalError%!(EXTRA string=String, string=Invalid Goby code)
			// ```
			//
			// @param Goby code [String]
			// @return [String]
			Name: "parse",
			Fn: func(receiver Object, sourceLine int) builtinMethodBody {
				return func(t *thread, args []Object, blockFrame *normalCallFrame) Object {
					if len(args) != 1 {
						return t.vm.initErrorObject(errors.ArgumentError, sourceLine, "Expect 1 argument. got=%d", len(args))
					}

					arg := args[0]
					switch arg.(type) {
					case *StringObject:
					default:
						return t.vm.initErrorObject(errors.TypeError, sourceLine, errors.WrongArgumentTypeFormat, classes.StringClass, arg.Class().Name)
					}

					l := lexer.New(arg.toString())
					p := parser.New(l)
					program, err := p.ParseProgram()

					if err != nil {
						return t.vm.initErrorObject(errors.TypeError, sourceLine, errors.InternalError, classes.StringClass, errors.InvalidGobyCode)
					}

					return t.vm.initStringObject(program.String())
				}
			},
		},
		{
			// Returns a tokenized Goby codes as an Array object.
			// Note that this does not return any errors even though the provided code is invalid.
			//
			// ```ruby
			// require 'ripper'; Ripper.token "10.times do |i| puts i end"
			// #=> ["10", ".", "times", "do", "|", "i", "|", "puts", "i", "end", "EOF"]
			//
			// require 'ripper'; Ripper.parse "10.times do |i| puts i" # the code is invalid
			// #=> ["10", ".", "times", "do", "|", "i", "|", "puts", "i", "EOF"]
			// ```
			//
			// @param Goby code [String]
			// @return [String]
			Name: "token",
			Fn: func(receiver Object, sourceLine int) builtinMethodBody {
				return func(t *thread, args []Object, blockFrame *normalCallFrame) Object {
					if len(args) != 1 {
						return t.vm.initErrorObject(errors.ArgumentError, sourceLine, "Expect 1 argument. got=%d", len(args))
					}

					arg := args[0]
					switch arg.(type) {
					case *StringObject:
					default:
						return t.vm.initErrorObject(errors.TypeError, sourceLine, errors.WrongArgumentTypeFormat, classes.StringClass, arg.Class().Name)
					}

					l := lexer.New(arg.toString())
					el := []Object{}
					var nt token.Token
					for i := 0; ; i++ {
						nt = l.NextToken()
						if nt.Type == token.EOF {
							el = append(el, t.vm.initStringObject("EOF"))
							break
						}
						el = append(el, t.vm.initStringObject(nt.Literal))
					}
					return t.vm.initArrayObject(el)
				}
			},
		},
	}
}

// Internal functions ===================================================
func initRipperClass(vm *VM) {
	rp := vm.initializeClass("Ripper", false)
	rp.setBuiltinMethods(builtInRipperClassMethods(), true)
	vm.objectClass.setClassConstant(rp)
}

// Other helper functions ----------------------------------------------

func (vm *VM) convertToTuple(instSet []*bytecode.InstructionSet) *ArrayObject {
	ary := []Object{}
	for _, insts := range instSet {
		hInsts := make(map[string]Object)
		hInsts["name"] = vm.initStringObject(insts.Name())
		hInsts["type"] = vm.initStringObject(insts.Type())
		if insts.ArgTypes() != nil {
			hInsts["arg_types"] = vm.getArgNameType(insts.ArgTypes())
		}
		ary = append(ary, vm.initHashObject(hInsts))

		aInst := []Object{}
		for _, ins := range insts.Instructions {
			hInst := make(map[string]Object)
			hInst["action"] = vm.initStringObject(ins.Action)
			hInst["line"] = vm.initIntegerObject(ins.Line())
			hInst["source_line"] = vm.initIntegerObject(ins.SourceLine())
			anchor, _ := ins.AnchorLine()
			hInst["anchor"] = vm.initIntegerObject(anchor)

			aParams := []Object{}
			for _, param := range ins.Params {
				aParams = append(aParams, vm.initStringObject(param))
			}
			hInst["params"] = vm.initArrayObject(aParams)

			if ins.ArgSet != nil {
				hInsts["arg_set"] = vm.getArgNameType(ins.ArgSet)
			}

			aInst = append(aInst, vm.initHashObject(hInst))
		}

		hInsts["instructions"] = vm.initArrayObject(aInst)
		ary = append(ary, vm.initHashObject(hInsts))
	}
	return vm.initArrayObject(ary)
}

func (vm *VM) getArgNameType(argSet *bytecode.ArgSet) *HashObject {
	h := make(map[string]Object)

	aName := []Object{}
	for _, argname := range argSet.Names() {
		aName = append(aName, vm.initStringObject(argname))
	}
	h["names"] = vm.initArrayObject(aName)

	aType := []Object{}
	for _, argtype := range argSet.Types() {
		aType = append(aType, vm.initIntegerObject(argtype))
	}

	h["types"] = vm.initArrayObject(aType)
	return vm.initHashObject(h)
}

// TODO: This should finally be auto-generated from token.go
func convertLex(t token.Type) string {
	var s string
	switch t {
	case token.Asterisk:
		s = "asterisk"
	case token.And:
		s = "and"
	case token.Assign:
		s = "assign"
	case token.Bang:
		s = "bang"
	case token.Bar:
		s = "bar"
	case token.Colon:
		s = "colon"
	case token.Comma:
		s = "comma"
	case token.COMP:
		s = "comp"
	case token.Dot:
		s = "dot"
	case token.Eq:
		s = "eq"
	case token.GT:
		s = "gt"
	case token.GTE:
		s = "gte"
	case token.LBrace:
		s = "lbrace"
	case token.LBracket:
		s = "lbracket"
	case token.LParen:
		s = "lparen"
	case token.LT:
		s = "lt"
	case token.LTE:
		s = "lte"
	case token.Match:
		s = "match"
	case token.Minus:
		s = "minus"
	case token.MinusEq:
		s = "minuseq"
	case token.Modulo:
		s = "modulo"
	case token.NotEq:
		s = "noteq"
	case token.Or:
		s = "or"
	case token.OrEq:
		s = "oreq"
	case token.Plus:
		s = "plus"
	case token.PlusEq:
		s = "pluseq"
	case token.Pow:
		s = "pow"
	case token.Range:
		s = "range"
	case token.RBrace:
		s = "rbrace"
	case token.RBracket:
		s = "rbracket"
	case token.ResolutionOperator:
		s = "resolutionoperator"
	case token.RParen:
		s = "rparen"
	case token.Semicolon:
		s = "semicolon"
	case token.Slash:
		s = "slash"
	default:
		s = strings.ToLower(string(t))
	}

	return "on_" + s
}
