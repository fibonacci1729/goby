package vm

import (
	"testing"
)

func TestRipperClassSuperclass(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`require 'ripper'; Ripper.class.name`, "Class"},
		{`require 'ripper'; Ripper.superclass.name`, "Object"},
		{`require 'ripper'; Ripper.ancestors.to_s`, "[Ripper, Object]"},
	}

	for i, tt := range tests {
		v := initTestVM()
		evaluated := v.testEval(t, tt.input, getFilename())
		verifyExpected(t, i, evaluated, tt.expected)
		v.checkCFP(t, i, 0)
		v.checkSP(t, i, 1)
	}
}

func TestRipperClassCreationFail(t *testing.T) {
	testsFail := []errorTestCase{
		{`require 'ripper'; Ripper.new`, "UnsupportedMethodError: Unsupported Method #new for Ripper", 1},
	}

	for i, tt := range testsFail {
		v := initTestVM()
		evaluated := v.testEval(t, tt.input, getFilename())
		checkErrorMsg(t, i, evaluated, tt.expected)
		v.checkCFP(t, i, tt.expectedCFP)
		v.checkSP(t, i, 1)
	}
}

func TestRipperParse(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`require 'ripper'; Ripper.parse "
	class Bar
		def self.foo
			10
		end
	end
	class Foo < Bar; end
	class FooBar < Foo; end
	FooBar.foo
"`, "class Bar {\ndef foo() {\n10\n}\n}class Foo {\n\n}class FooBar {\n\n}FooBar.foo()"},
		{`require 'ripper'; Ripper.parse "
	def foo(x)
	  yield(x + 10)
	end
	def bar(y)
	  foo(y) do |f|
		yield(f)
	  end
	end
	def baz(z)
	  bar(z + 100) do |b|
		yield(b)
	  end
	end
	a = 0
	baz(100) do |b|
	  a = b
	end
	a

	class Foo
	  def bar
		100
	  end
	end
	module Baz
	  class Bar
		def bar
		  Foo.new.bar
		end
	  end
	end
	Baz::Bar.new.bar + a
"`, "def foo(x) {\nyield((x + 10))\n}def bar(y) {\nself.foo(y) do |f|\nyield(f)\nend\n}def baz(z) {\nself.bar((z + 100)) do |b|\nyield(b)\nend\n}a = 0self.baz(100) do |b|\na = b\nendaclass Foo {\ndef bar() {\n100\n}\n}module Baz {\nclass Bar {\ndef bar() {\nFoo.new().bar()\n}\n}\n}((Baz :: Bar).new().bar() + a)"},
		{`require 'ripper'; Ripper.parse "
	def bar(block)
	block.call + get_block.call
	end
	
	def foo
		bar(get_block) do
  		20
		end
	end
	
	foo do
		10
	end
"`, "def bar(block) {\n(block.call() + get_block.call())\n}def foo() {\nself.bar(get_block) do\n20\nend\n}self.foo() do\n10\nend"},
	}

	for i, tt := range tests {
		v := initTestVM()
		evaluated := v.testEval(t, tt.input, getFilename())
		verifyExpected(t, i, evaluated, tt.expected)
		v.checkCFP(t, i, 0)
		v.checkSP(t, i, 1)
	}
}

func TestRipperParseFail(t *testing.T) {
	testsFail := []errorTestCase{
		{`require 'ripper'; Ripper.parse`, "ArgumentError: Expect 1 argument. got=0", 1},
		{`require 'ripper'; Ripper.parse(1)`, "TypeError: Expect argument to be String. got: Integer", 1},
		{`require 'ripper'; Ripper.parse(1.2)`, "TypeError: Expect argument to be String. got: Float", 1},
		{`require 'ripper'; Ripper.parse(["puts", "123"])`, "TypeError: Expect argument to be String. got: Array", 1},
		{`require 'ripper'; Ripper.parse({key: 1})`, "TypeError: Expect argument to be String. got: Hash", 1},
	}

	for i, tt := range testsFail {
		v := initTestVM()
		evaluated := v.testEval(t, tt.input, getFilename())
		checkErrorMsg(t, i, evaluated, tt.expected)
		v.checkCFP(t, i, tt.expectedCFP)
		v.checkSP(t, i, 1)
	}
}

func TestRipperToken(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`require 'ripper'; Ripper.token("
	class Bar
		def self.foo
			10
		end
	end
	class Foo < Bar; end
	class FooBar < Foo; end
	FooBar.foo
").to_s`, `["class", "Bar", "def", "self", ".", "foo", "10", "end", "end", "class", "Foo", "<", "Bar", ";", "end", "class", "FooBar", "<", "Foo", ";", "end", "FooBar", ".", "foo", "EOF"]`},
		{`require 'ripper'; Ripper.token("
	def foo(x)
	  yield(x + 10)
	end
	def bar(y)
	  foo(y) do |f|
		yield(f)
	  end
	end
	def baz(z)
	  bar(z + 100) do |b|
		yield(b)
	  end
	end
	a = 0
	baz(100) do |b|
	  a = b
	end
	a

	class Foo
	  def bar
		100
	  end
	end
	module Baz
	  class Bar
		def bar
		  Foo.new.bar
		end
	  end
	end
	Baz::Bar.new.bar + a
").to_s`, `["def", "foo", "(", "x", ")", "yield", "(", "x", "+", "10", ")", "end", "def", "bar", "(", "y", ")", "foo", "(", "y", ")", "do", "|", "f", "|", "yield", "(", "f", ")", "end", "end", "def", "baz", "(", "z", ")", "bar", "(", "z", "+", "100", ")", "do", "|", "b", "|", "yield", "(", "b", ")", "end", "end", "a", "=", "0", "baz", "(", "100", ")", "do", "|", "b", "|", "a", "=", "b", "end", "a", "class", "Foo", "def", "bar", "100", "end", "end", "module", "Baz", "class", "Bar", "def", "bar", "Foo", ".", "new", ".", "bar", "end", "end", "end", "Baz", "::", "Bar", ".", "new", ".", "bar", "+", "a", "EOF"]`},
		{`require 'ripper'; Ripper.token("
	def bar(block)
	block.call + get_block.call
	end
	
	def foo
		bar(get_block) do
  		20
		end
	end
	
	foo do
		10
	end
").to_s`, `["def", "bar", "(", "block", ")", "block", ".", "call", "+", "get_block", ".", "call", "end", "def", "foo", "bar", "(", "get_block", ")", "do", "20", "end", "end", "foo", "do", "10", "end", "EOF"]`},
	}

	for i, tt := range tests {
		v := initTestVM()
		evaluated := v.testEval(t, tt.input, getFilename())
		verifyExpected(t, i, evaluated, tt.expected)
		v.checkCFP(t, i, 0)
		v.checkSP(t, i, 1)
	}
}

func TestRipperTokenFail(t *testing.T) {
	testsFail := []errorTestCase{
		{`require 'ripper'; Ripper.token`, "ArgumentError: Expect 1 argument. got=0", 1},
		{`require 'ripper'; Ripper.token(1)`, "TypeError: Expect argument to be String. got: Integer", 1},
		{`require 'ripper'; Ripper.token(1.2)`, "TypeError: Expect argument to be String. got: Float", 1},
		{`require 'ripper'; Ripper.token(["puts", "123"])`, "TypeError: Expect argument to be String. got: Array", 1},
		{`require 'ripper'; Ripper.token({key: 1})`, "TypeError: Expect argument to be String. got: Hash", 1},
	}

	for i, tt := range testsFail {
		v := initTestVM()
		evaluated := v.testEval(t, tt.input, getFilename())
		checkErrorMsg(t, i, evaluated, tt.expected)
		v.checkCFP(t, i, tt.expectedCFP)
		v.checkSP(t, i, 1)
	}
}

func TestRipperLex(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`require 'ripper'; Ripper.lex("
	class Bar
		def self.foo
			10
		end
	end
	class Foo < Bar; end
	class FooBar < Foo; end
	FooBar.foo
").to_s`, `[[1, "on_class", "class"], [1, "on_constant", "Bar"], [2, "on_def", "def"], [2, "on_self", "self"], [2, "on_dot", "."], [2, "on_ident", "foo"], [3, "on_int", "10"], [4, "on_end", "end"], [5, "on_end", "end"], [6, "on_class", "class"], [6, "on_constant", "Foo"], [6, "on_lt", "<"], [6, "on_constant", "Bar"], [6, "on_semicolon", ";"], [6, "on_end", "end"], [7, "on_class", "class"], [7, "on_constant", "FooBar"], [7, "on_lt", "<"], [7, "on_constant", "Foo"], [7, "on_semicolon", ";"], [7, "on_end", "end"], [8, "on_constant", "FooBar"], [8, "on_dot", "."], [8, "on_ident", "foo"], [9, "on_eof", ""]]`},
		{`require 'ripper'; Ripper.lex("
	def foo(x)
	  yield(x + 10)
	end
	def bar(y)
	  foo(y) do |f|
		yield(f)
	  end
	end
	def baz(z)
	  bar(z + 100) do |b|
		yield(b)
	  end
	end
	a = 0
	baz(100) do |b|
	  a = b
	end
	a

	class Foo
	  def bar
		100
	  end
	end
	module Baz
	  class Bar
		def bar
		  Foo.new.bar
		end
	  end
	end
	Baz::Bar.new.bar + a
").to_s`, `[[1, "on_def", "def"], [1, "on_ident", "foo"], [1, "on_lparen", "("], [1, "on_ident", "x"], [1, "on_rparen", ")"], [2, "on_yield", "yield"], [2, "on_lparen", "("], [2, "on_ident", "x"], [2, "on_plus", "+"], [2, "on_int", "10"], [2, "on_rparen", ")"], [3, "on_end", "end"], [4, "on_def", "def"], [4, "on_ident", "bar"], [4, "on_lparen", "("], [4, "on_ident", "y"], [4, "on_rparen", ")"], [5, "on_ident", "foo"], [5, "on_lparen", "("], [5, "on_ident", "y"], [5, "on_rparen", ")"], [5, "on_do", "do"], [5, "on_bar", "|"], [5, "on_ident", "f"], [5, "on_bar", "|"], [6, "on_yield", "yield"], [6, "on_lparen", "("], [6, "on_ident", "f"], [6, "on_rparen", ")"], [7, "on_end", "end"], [8, "on_end", "end"], [9, "on_def", "def"], [9, "on_ident", "baz"], [9, "on_lparen", "("], [9, "on_ident", "z"], [9, "on_rparen", ")"], [10, "on_ident", "bar"], [10, "on_lparen", "("], [10, "on_ident", "z"], [10, "on_plus", "+"], [10, "on_int", "100"], [10, "on_rparen", ")"], [10, "on_do", "do"], [10, "on_bar", "|"], [10, "on_ident", "b"], [10, "on_bar", "|"], [11, "on_yield", "yield"], [11, "on_lparen", "("], [11, "on_ident", "b"], [11, "on_rparen", ")"], [12, "on_end", "end"], [13, "on_end", "end"], [14, "on_ident", "a"], [14, "on_assign", "="], [14, "on_int", "0"], [15, "on_ident", "baz"], [15, "on_lparen", "("], [15, "on_int", "100"], [15, "on_rparen", ")"], [15, "on_do", "do"], [15, "on_bar", "|"], [15, "on_ident", "b"], [15, "on_bar", "|"], [16, "on_ident", "a"], [16, "on_assign", "="], [16, "on_ident", "b"], [17, "on_end", "end"], [18, "on_ident", "a"], [20, "on_class", "class"], [20, "on_constant", "Foo"], [21, "on_def", "def"], [21, "on_ident", "bar"], [22, "on_int", "100"], [23, "on_end", "end"], [24, "on_end", "end"], [25, "on_module", "module"], [25, "on_constant", "Baz"], [26, "on_class", "class"], [26, "on_constant", "Bar"], [27, "on_def", "def"], [27, "on_ident", "bar"], [28, "on_constant", "Foo"], [28, "on_dot", "."], [28, "on_ident", "new"], [28, "on_dot", "."], [28, "on_ident", "bar"], [29, "on_end", "end"], [30, "on_end", "end"], [31, "on_end", "end"], [32, "on_constant", "Baz"], [32, "on_resolutionoperator", "::"], [32, "on_constant", "Bar"], [32, "on_dot", "."], [32, "on_ident", "new"], [32, "on_dot", "."], [32, "on_ident", "bar"], [32, "on_plus", "+"], [32, "on_ident", "a"], [33, "on_eof", ""]]`},
		{`require 'ripper'; Ripper.lex("
	def bar(block)
	block.call + get_block.call
	end
	
	def foo
		bar(get_block) do
  		20
		end
	end
	
	foo do
		10
	end
").to_s`, `[[1, "on_def", "def"], [1, "on_ident", "bar"], [1, "on_lparen", "("], [1, "on_ident", "block"], [1, "on_rparen", ")"], [2, "on_ident", "block"], [2, "on_dot", "."], [2, "on_ident", "call"], [2, "on_plus", "+"], [2, "on_get_block", "get_block"], [2, "on_dot", "."], [2, "on_ident", "call"], [3, "on_end", "end"], [5, "on_def", "def"], [5, "on_ident", "foo"], [6, "on_ident", "bar"], [6, "on_lparen", "("], [6, "on_get_block", "get_block"], [6, "on_rparen", ")"], [6, "on_do", "do"], [7, "on_int", "20"], [8, "on_end", "end"], [9, "on_end", "end"], [11, "on_ident", "foo"], [11, "on_do", "do"], [12, "on_int", "10"], [13, "on_end", "end"], [14, "on_eof", ""]]`},
	}

	for i, tt := range tests {
		v := initTestVM()
		evaluated := v.testEval(t, tt.input, getFilename())
		verifyExpected(t, i, evaluated, tt.expected)
		v.checkCFP(t, i, 0)
		v.checkSP(t, i, 1)
	}
}

func TestRipperLexFail(t *testing.T) {
	testsFail := []errorTestCase{
		{`require 'ripper'; Ripper.lex`, "ArgumentError: Expect 1 argument. got=0", 1},
		{`require 'ripper'; Ripper.lex(1)`, "TypeError: Expect argument to be String. got: Integer", 1},
		{`require 'ripper'; Ripper.lex(1.2)`, "TypeError: Expect argument to be String. got: Float", 1},
		{`require 'ripper'; Ripper.lex(["puts", "123"])`, "TypeError: Expect argument to be String. got: Array", 1},
		{`require 'ripper'; Ripper.lex({key: 1})`, "TypeError: Expect argument to be String. got: Hash", 1},
	}

	for i, tt := range testsFail {
		v := initTestVM()
		evaluated := v.testEval(t, tt.input, getFilename())
		checkErrorMsg(t, i, evaluated, tt.expected)
		v.checkCFP(t, i, tt.expectedCFP)
		v.checkSP(t, i, 1)
	}
}
