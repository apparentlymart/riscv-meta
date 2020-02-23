package main

type MajorOpcode struct {
	Name     string
	FuncName string
	TypeName string
	Num      bits8
}

type Codec struct {
	Name     string
	FuncName string
	TypeName string
	Operands []string
}

type Operation struct {
	FullName    string
	Description string
	Pseudocode  string
	Name        string
	FuncName    string
	TypeName    string
	MajorOpcode *MajorOpcode
	Codec       *Codec
	Test, Mask  bits32
	Standards   Standards
}

type Argument struct {
	Name          string
	FuncName      string
	TypeName      string
	FuncLocalName string
	TypeLocalName string
	Type          ArgType
	Decoding      []ArgDecodeStep
}

type ISA struct {
	ExtensionNames map[Extension]string
	MajorOpcodes   map[bits8]*MajorOpcode
	Codecs         map[string]*Codec
	Arguments      map[string]*Argument
	Expansions     map[string]string
	Ops            []Operation
}
