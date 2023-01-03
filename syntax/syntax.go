// package syntax ...
//
// [inspired|forked] implementation of [github.com/sourcegraph/syntaxhighlight]
//
// This is a [minimal|static|optimize] fork of the great [github.com/sourcegraph/syntaxhighligh]
// this fork is adapted and optimized for an specific use-case and *is not* api/result
// compatible with the original. Please do not use outside very specific use cases!.
//
// Please use always the original!
//
// Copyright (c) 2013, Sourcegraph, Inc. - All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// Redistributions of source code must retain the above copyright notice,
// this list of conditions and the following disclaimer. Redistributions
// in binary form must reproduce the above copyright notice, this list of
// conditions and the following disclaimer in the documentation and/or
// other materials provided with the distribution.
//
// Neither the name of Sourcegraph nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package syntax

import (
	"bytes"
	"io"
	"strings"
	"text/scanner"
	"text/template"
	"unicode"
	"unicode/utf8"
)

// Kind ...
type Kind uint8

// const
const (
	Whitespace Kind = iota
	String
	Keyword
	Comment
	Type
	Literal
	Punctuation
	Plaintext
	Tag
	HTMLTag
	HTMLAttrName
	HTMLAttrValue
	Decimal
)

// Printer ...
//
//go:generate gostringer -type=Kind
type Printer interface {
	Print(w io.Writer, kind Kind, tokText string) error
}

// HTMLConfig ...
type HTMLConfig struct {
	String        string
	Keyword       string
	Comment       string
	Type          string
	Literal       string
	Punctuation   string
	Plaintext     string
	Tag           string
	HTMLTag       string
	HTMLAttrName  string
	HTMLAttrValue string
	Decimal       string
	Whitespace    string
	AsOrderedList bool
}

// HTMLPrinter ...
type HTMLPrinter HTMLConfig

// Class ....
func (c HTMLConfig) Class(kind Kind) string {
	switch kind {
	case String:
		return c.String
	case Keyword:
		return c.Keyword
	case Comment:
		return c.Comment
	case Type:
		return c.Type
	case Literal:
		return c.Literal
	case Punctuation:
		return c.Punctuation
	case Plaintext:
		return c.Plaintext
	case Tag:
		return c.Tag
	case HTMLTag:
		return c.HTMLTag
	case HTMLAttrName:
		return c.HTMLAttrName
	case HTMLAttrValue:
		return c.HTMLAttrValue
	case Decimal:
		return c.Decimal
	}
	return ""
}

// Print ...
func (p HTMLPrinter) Print(w io.Writer, kind Kind, tokText string) error {
	if p.AsOrderedList {
		if i := strings.Index(tokText, "\n"); i > -1 {
			if err := p.Print(w, kind, tokText[:i]); err != nil {
				return err
			}
			if err := w.Write([]byte("</li>\n\t\t\t<li>")); err != nil {
				return err
			}
			if err := p.Print(w, kind, tokText[i+1:]); err != nil {
				return err
			}
			return nil
		}
	}

	class := ((HTMLConfig)(p)).Class(kind)
	if class != "" {
		_, err := w.Write([]byte(`<span class="`))
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, class)
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(`">`))
		if err != nil {
			return err
		}
	}
	template.HTMLEscape(w, []byte(tokText))
	if class != "" {
		_, err := w.Write([]byte(`</span>`))
		if err != nil {
			return err
		}
	}
	return nil
}

// Option ...
type Option func(options *HTMLConfig)

// OrderedList ...
func OrderedList() Option {
	return func(o *HTMLConfig) {
		o.AsOrderedList = true
	}
}

// DefaultHTMLConfig ...
var DefaultHTMLConfig = HTMLConfig{
	String: "str", Keyword: "kwd", Comment: "com", Type: "typ", Literal: "lit", Punctuation: "pun", Plaintext: "pln",
	Tag: "tag", HTMLTag: "htm", HTMLAttrName: "atn", HTMLAttrValue: "atv", Decimal: "dec", Whitespace: "",
}

// Print ...
func Print(s *scanner.Scanner, w io.Writer, p Printer) error {
	tok := s.Scan()
	for tok != scanner.EOF {
		tokText := s.TokenText()
		err := p.Print(w, tokenKind(tok, tokText), tokText)
		if err != nil {
			return err
		}

		tok = s.Scan()
	}

	return nil
}

// AsHTML ...
func AsHTML(src []byte, options ...Option) ([]byte, error) {
	opt := DefaultHTMLConfig
	for _, f := range options {
		f(&opt)
	}

	var buf bytes.Buffer
	if opt.AsOrderedList {
		buf.Write([]byte("\t\t\t<ol><li>"))
	}
	err := Print(NewScanner(src), &buf, HTMLPrinter(opt))
	if opt.AsOrderedList {
		buf.Write([]byte("</li>\n\t\t\t</ol>"))
	}
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// NewScanner ...
func NewScanner(src []byte) *scanner.Scanner {
	return NewScannerReader(bytes.NewReader(src))
}

// NewScannerReader ...
func NewScannerReader(src io.Reader) *scanner.Scanner {
	var s scanner.Scanner
	s.Init(src)
	s.Error = func(_ *scanner.Scanner, _ string) {}
	s.Whitespace = 0
	s.Mode = s.Mode ^ scanner.SkipComments
	return &s
}

func tokenKind(tok rune, tokText string) Kind {
	switch tok {
	case scanner.Ident:
		if _, isKW := keywords[tokText]; isKW {
			return Keyword
		}
		if r, _ := utf8.DecodeRuneInString(tokText); unicode.IsUpper(r) {
			return Type
		}
		return Plaintext
	case scanner.Float, scanner.Int:
		return Decimal
	case scanner.Char, scanner.String, scanner.RawString:
		return String
	case scanner.Comment:
		return Comment
	}
	if unicode.IsSpace(tok) {
		return Whitespace
	}
	return Punctuation
}

var keywords = map[string]struct{}{
	"---":              {},
	"+++":              {},
	"Accept":           {},
	"User-Agent":       {},
	"Connection":       {},
	"Sec-Fetch":        {},
	"Cache":            {},
	"Upgrade":          {},
	"Fetch":            {},
	"Encoding":         {},
	"BEGIN":            {},
	"END":              {},
	"False":            {},
	"Infinity":         {},
	"NaN":              {},
	"None":             {},
	"True":             {},
	"abstract":         {},
	"alias":            {},
	"align_union":      {},
	"alignof":          {},
	"and":              {},
	"append":           {},
	"as":               {},
	"asm":              {},
	"assert":           {},
	"auto":             {},
	"any":              {},
	"axiom":            {},
	"begin":            {},
	"bool":             {},
	"boolean":          {},
	"break":            {},
	"byte":             {},
	"caller":           {},
	"case":             {},
	"catch":            {},
	"char":             {},
	"class":            {},
	"concept":          {},
	"concept_map":      {},
	"const":            {},
	"const_cast":       {},
	"constexpr":        {},
	"continue":         {},
	"debugger":         {},
	"decltype":         {},
	"def":              {},
	"default":          {},
	"defined":          {},
	"del":              {},
	"delegate":         {},
	"delete":           {},
	"die":              {},
	"diff":             {},
	"do":               {},
	"double":           {},
	"dump":             {},
	"dynamic_cast":     {},
	"elif":             {},
	"else":             {},
	"elsif":            {},
	"end":              {},
	"ensure":           {},
	"enum":             {},
	"err":              {},
	"error":            {},
	"eval":             {},
	"except":           {},
	"exec":             {},
	"exit":             {},
	"explicit":         {},
	"export":           {},
	"extends":          {},
	"extern":           {},
	"false":            {},
	"final":            {},
	"finally":          {},
	"float":            {},
	"float32":          {},
	"float64":          {},
	"for":              {},
	"foreach":          {},
	"friend":           {},
	"from":             {},
	"func":             {},
	"function":         {},
	"generic":          {},
	"get":              {},
	"global":           {},
	"goto":             {},
	"if":               {},
	"implements":       {},
	"import":           {},
	"in":               {},
	"index":            {},
	"inline":           {},
	"instanceof":       {},
	"int":              {},
	"int8":             {},
	"int16":            {},
	"int32":            {},
	"int64":            {},
	"interface":        {},
	"is":               {},
	"lambda":           {},
	"last":             {},
	"late_check":       {},
	"local":            {},
	"long":             {},
	"make":             {},
	"map":              {},
	"module":           {},
	"mutable":          {},
	"my":               {},
	"namespace":        {},
	"native":           {},
	"new":              {},
	"next":             {},
	"nil":              {},
	"no":               {},
	"nonlocal":         {},
	"not":              {},
	"null":             {},
	"nullptr":          {},
	"operator":         {},
	"or":               {},
	"our":              {},
	"package":          {},
	"pass":             {},
	"print":            {},
	"private":          {},
	"property":         {},
	"protected":        {},
	"public":           {},
	"raise":            {},
	"redo":             {},
	"register":         {},
	"reinterpret_cast": {},
	"require":          {},
	"rescue":           {},
	"retry":            {},
	"return":           {},
	"self":             {},
	"set":              {},
	"short":            {},
	"signed":           {},
	"sizeof":           {},
	"static":           {},
	"static_assert":    {},
	"static_cast":      {},
	"strictfp":         {},
	"string":           {},
	"struct":           {},
	"sub":              {},
	"super":            {},
	"switch":           {},
	"synchronized":     {},
	"template":         {},
	"then":             {},
	"this":             {},
	"throw":            {},
	"throws":           {},
	"transient":        {},
	"true":             {},
	"try":              {},
	"type":             {},
	"typedef":          {},
	"typeid":           {},
	"typename":         {},
	"typeof":           {},
	"undef":            {},
	"undefined":        {},
	"union":            {},
	"unless":           {},
	"unsigned":         {},
	"until":            {},
	"use":              {},
	"using":            {},
	"var":              {},
	"virtual":          {},
	"void":             {},
	"volatile":         {},
	"wantarray":        {},
	"when":             {},
	"where":            {},
	"while":            {},
	"with":             {},
	"yield":            {},
}
