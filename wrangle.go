package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/davecgh/go-spew/spew"
)

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
	MajorOpcodes map[bits8]*MajorOpcode
	Codecs       map[string]*Codec
	Arguments    map[string]*Argument
	Ops          []Operation
	Expansions   map[string]string
}

type Extension byte
type Size uint8
type Standard uint16
type Standards map[Standard]struct{}

const (
	RVInvalid Size = 0
	RV32      Size = 32
	RV64      Size = 64
	RV128     Size = 128
)

const (
	ExtInvalid Extension = 0
	ExtI       Extension = 'I' // base integer
	ExtM       Extension = 'M' // multiply and divide
	ExtA       Extension = 'A' // atomic
	ExtS       Extension = 'S' // supervisor
	ExtF       Extension = 'F' // single-precision floating point
	ExtD       Extension = 'D' // double-precision floating point
	ExtQ       Extension = 'Q' // quad-precision floating point
	ExtC       Extension = 'C' // compressed
)

const (
	Invalid = Standard(0)

	RV32Any = Standard(uint16(RV32))
	RV32I   = Standard(uint16(RV32) | uint16(ExtI)<<8)
	RV32M   = Standard(uint16(RV32) | uint16(ExtM)<<8)
	RV32A   = Standard(uint16(RV32) | uint16(ExtA)<<8)
	RV32S   = Standard(uint16(RV32) | uint16(ExtS)<<8)
	RV32F   = Standard(uint16(RV32) | uint16(ExtF)<<8)
	RV32D   = Standard(uint16(RV32) | uint16(ExtD)<<8)
	RV32Q   = Standard(uint16(RV32) | uint16(ExtQ)<<8)
	RV32C   = Standard(uint16(RV32) | uint16(ExtC)<<8)

	RV64Any = Standard(uint16(RV64))
	RV64I   = Standard(uint16(RV64) | uint16(ExtI)<<8)
	RV64M   = Standard(uint16(RV64) | uint16(ExtM)<<8)
	RV64A   = Standard(uint16(RV64) | uint16(ExtA)<<8)
	RV64S   = Standard(uint16(RV64) | uint16(ExtS)<<8)
	RV64F   = Standard(uint16(RV64) | uint16(ExtF)<<8)
	RV64D   = Standard(uint16(RV64) | uint16(ExtD)<<8)
	RV64Q   = Standard(uint16(RV64) | uint16(ExtQ)<<8)
	RV64C   = Standard(uint16(RV64) | uint16(ExtC)<<8)

	RV128Any = Standard(uint16(RV128))
	RV128I   = Standard(uint16(RV128) | uint16(ExtI)<<8)
	RV128M   = Standard(uint16(RV128) | uint16(ExtM)<<8)
	RV128A   = Standard(uint16(RV128) | uint16(ExtA)<<8)
	RV128S   = Standard(uint16(RV128) | uint16(ExtS)<<8)
	RV128F   = Standard(uint16(RV128) | uint16(ExtF)<<8)
	RV128D   = Standard(uint16(RV128) | uint16(ExtD)<<8)
	RV128Q   = Standard(uint16(RV128) | uint16(ExtQ)<<8)
	RV128C   = Standard(uint16(RV128) | uint16(ExtC)<<8)
)

func (s Standard) Size() Size {
	return Size(s & 0xff)
}

func (s Standard) Extension() Extension {
	return Extension(s >> 8)
}

func (s Standard) Base() Standard {
	return Standard(s & 0xff)
}

func (s Standard) String() string {
	size := s.Size()
	ext := s.Extension()
	if ext == ExtInvalid {
		return fmt.Sprintf("RV%d", size)
	}
	return fmt.Sprintf("RV%d%c", size, ext)
}

func (ss Standards) Has(s Standard) bool {
	_, ok := ss[s]
	return ok
}

func (ss Standards) Add(s Standard) {
	ss[s] = struct{}{}
}

func (ss Standards) String() string {
	var ssList []Standard
	for s := range ss {
		ssList = append(ssList, s)
	}
	sort.Slice(ssList, func(i, j int) bool {
		return ssList[i] < ssList[j]
	})
	var buf strings.Builder
	for i, s := range ssList {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(s.String())
	}
	return buf.String()
}

func ParseStandard(s string) Standard {
	if !strings.HasPrefix(s, "rv") {
		return Invalid
	}
	bitsStr := s[2 : len(s)-1]
	var bits Size
	switch bitsStr {
	case "32":
		bits = RV32
	case "64":
		bits = RV64
	case "128":
		bits = RV128
	default:
		return Invalid
	}
	ext := Extension(strings.ToUpper(string(s[len(s)-1]))[0])

	return Standard(uint16(bits) | uint16(ext)<<8)
}

type ArgType string

const (
	ArgGeneral           ArgType = "arg"
	ArgIntReg            ArgType = "ireg"
	ArgFloatReg          ArgType = "freg"
	ArgCompressedReg     ArgType = "creg"
	ArgOffset            ArgType = "offset"
	ArgSignedImmediate   ArgType = "simm"
	ArgUnsignedImmediate ArgType = "uimm"
)

type ArgDecodeStep struct {
	Mask      bits32
	LeftShift int
}

type bits8 uint8
type bits32 uint32

func (v bits8) String() string {
	return fmt.Sprintf("0b%08b", v)
}

func (v bits32) String() string {
	return fmt.Sprintf("0b%032b", v)
}

func loadMajorOpcodes(filename string) (map[bits8]*MajorOpcode, error) {
	r, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	ret := make(map[bits8]*MajorOpcode)

	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := trimComments(sc.Text())
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := fields[len(fields)-1]
		fields = fields[:len(fields)-1]

		// Only the "real" (currently assigned) opcodes are all uppercase,
		// so we'll use that as a heuristic to filter out all the others
		// that mark coding space reservations.
		if strings.ToUpper(name) != name {
			continue
		}

		oc := &MajorOpcode{
			Name:     name,
			FuncName: makeIdentUnderscores(name),
			TypeName: makeIdentTitle(name),
			Num:      0b11, // two low-order bytes are always set for these 32-bit major opcodes
		}

		for _, rawSpec := range fields {
			v, _ := parseMatchSpec(rawSpec)
			oc.Num |= bits8(v)
		}

		ret[oc.Num] = oc
	}

	return ret, nil
}

func loadCodecs(filename string) (map[string]*Codec, error) {
	r, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	ret := make(map[string]*Codec)

	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := trimComments(sc.Text())
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := fields[0]

		cd := &Codec{
			Name:     name,
			FuncName: makeIdentUnderscores(name),
			TypeName: makeIdentTitle(name),
			Operands: fields[2:],
		}

		ret[cd.Name] = cd
	}

	return ret, sc.Err()
}

func loadArgs(filename string) (map[string]*Argument, error) {
	r, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	ret := make(map[string]*Argument)

	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := trimComments(sc.Text())
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		name := fields[0]

		arg := &Argument{
			Name:          name,
			FuncName:      makeIdentUnderscores(name),
			TypeName:      makeIdentTitle(name),
			FuncLocalName: strings.ReplaceAll(makeIdentUnderscores(fields[3]), "_", ""),
			TypeLocalName: makeIdentTitle(fields[3]),
			Type:          ArgType(fields[2]),
			// TODO: The decoding steps
		}

		ret[name] = arg
	}

	return ret, nil
}

func loadOperations(filename string, majors map[bits8]*MajorOpcode, codecs map[string]*Codec, fullNames map[string]string, descs map[string]string, pseudocode map[string]string) ([]Operation, error) {
	r, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	var ret []Operation

	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := trimComments(sc.Text())
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		name := fields[0]
		fields = fields[1:] // skip name

		op := Operation{
			FullName:    fullNames[name],
			Description: descs[name],
			Pseudocode:  pseudocode[name],
			Name:        name,
			FuncName:    makeIdentUnderscores(name),
			TypeName:    makeIdentTitle(name),

			Standards: make(Standards),
		}

		// The fields after the name are a mixture of field names and
		// matching specs until we find a codec name. We don't actually
		// need the field names (they are implied by the codec), so we'll
		// skip over them.
		for len(fields) > 0 {
			rawMatch := fields[0]
			fields = fields[1:]

			if codec, ok := codecs[rawMatch]; ok {
				// If we've found a codec then we've reached the end of
				// the first variable-length portion, and we've found our
				// codec identifier too.
				op.Codec = codec
				break
			}
			if !unicode.IsDigit(rune(rawMatch[0])) {
				// not a matching spec, then
				continue
			}

			v, mask := parseMatchSpec(rawMatch)
			op.Test |= bits32(v)
			op.Mask |= bits32(mask)
		}

		// If we get here without having a codec set then the line must be
		// invalid, so we'll just skip it.
		if op.Codec == nil {
			continue
		}

		// If it's a standard-length instruction (as opposed to compressed
		// or extended length) then we'll find the major opcode it belongs
		// to, which an instruction decoder can use to partition the coding
		// space rather than scanning over all of the operations every time.
		if (op.Mask & 0b1111111) == 0b1111111 {
			majorOpcode := bits8(op.Test & 0b1111111)
			op.MajorOpcode = majors[majorOpcode]
		}

		// Any remaining fields should be standards identifiers indicating
		// which standard(s) this operation belongs to. Note that operation
		// names are unique only within a particular architecture "size"
		// (RV32, RV64, or RV128).
		for _, raw := range fields {
			std := ParseStandard(raw)
			op.Standards.Add(std)
			op.Standards.Add(std.Base())
		}

		ret = append(ret, op)
	}

	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Name < ret[j].Name
	})

	return ret, sc.Err()
}

func loadExpansions(filename string) (map[string]string, error) {
	r, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	ret := make(map[string]string)

	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := trimComments(sc.Text())
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		ret[fields[0]] = fields[1]
	}

	return ret, nil
}

func loadOpcodeStrings(filename string) (map[string]string, error) {
	r, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	ret := make(map[string]string)

	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := trimComments(sc.Text())
		quot := strings.IndexRune(line, '"')
		if quot < 0 {
			continue
		}
		mnem := strings.TrimSpace(line[:quot])
		str := line[quot+1:]
		quot = strings.IndexRune(str, '"')
		if quot >= 0 {
			str = str[:quot]
		}

		ret[mnem] = strings.TrimSpace(str)
	}

	return ret, sc.Err()
}

func loadISAMeta() (*ISA, error) {
	majorOpcodes, err := loadMajorOpcodes("opcode-majors")
	if err != nil {
		return nil, fmt.Errorf("failed to load major opcodes: %s", err)
	}
	codecs, err := loadCodecs("codecs")
	if err != nil {
		return nil, fmt.Errorf("failed to load codecs: %s", err)
	}
	args, err := loadArgs("operands")
	if err != nil {
		return nil, fmt.Errorf("failed to load operands: %s", err)
	}
	opFullNames, err := loadOpcodeStrings("opcode-fullnames")
	if err != nil {
		return nil, fmt.Errorf("failed to load operation full names: %s", err)
	}
	opDescs, err := loadOpcodeStrings("opcode-descriptions")
	if err != nil {
		return nil, fmt.Errorf("failed to load operation descriptions: %s", err)
	}
	opPseudocode, err := loadOpcodeStrings("opcode-pseudocode-alt")
	if err != nil {
		return nil, fmt.Errorf("failed to load operation pseudocode: %s", err)
	}
	ops, err := loadOperations("opcodes", majorOpcodes, codecs, opFullNames, opDescs, opPseudocode)
	if err != nil {
		return nil, fmt.Errorf("failed to load minor opcodes: %s", err)
	}
	exps, err := loadExpansions("compression")
	if err != nil {
		return nil, fmt.Errorf("failed to load compressed opcode expansion table: %s", err)
	}

	return &ISA{
		MajorOpcodes: majorOpcodes,
		Codecs:       codecs,
		Arguments:    args,
		Ops:          ops,
		Expansions:   exps,
	}, nil
}

func main() {
	isa, err := loadISAMeta()
	if err != nil {
		log.Fatal(err)
	}

	spew.Dump(isa)
}

func trimComments(line string) string {
	hash := strings.IndexByte(line, '#')
	if hash == -1 {
		return line
	}
	return line[:hash]
}

func makeIdentUnderscores(inp string) string {
	var b strings.Builder
	for i, r := range inp {
		switch {
		case unicode.IsDigit(r):
			if i == 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r)
		case unicode.IsLetter(r):
			b.WriteString(strings.ToLower(string(r)))
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

func makeIdentTitle(inp string) string {
	var b strings.Builder
	nextUpper := true
	for i, r := range inp {
		switch {
		case unicode.IsDigit(r):
			if i == 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r)
			nextUpper = true
		case unicode.IsLetter(r):
			if nextUpper {
				b.WriteString(strings.ToUpper(string(r)))
			} else {
				b.WriteString(strings.ToLower(string(r)))
			}
			nextUpper = false
		default:
			nextUpper = true
		}
	}
	return b.String()
}

func partition(s string, sep string) (l, r string) {
	idx := strings.Index(s, sep)
	if idx == -1 {
		return s, ""
	}
	return s[:idx], s[idx+len(sep):]
}

func parseMatchSpec(rawSpec string) (val uint32, mask uint32) {
	rawRng, rawWant := partition(rawSpec, "=")
	want, err := strconv.ParseUint(rawWant, 0, 32)
	if err != nil {
		return 0, 0
	}
	rawEnd, rawStart := partition(rawRng, "..")
	start, err := strconv.ParseUint(rawStart, 10, 64)
	if err != nil {
		return 0, 0
	}
	end, err := strconv.ParseUint(rawEnd, 10, 64)
	if err != nil {
		return 0, 0
	}
	mask = uint32((1 << (end + 1)) - (1 << start))

	// We're just assuming that there won't be a "val" that is too
	// big to fit in the identified bits here, which means we can ignore
	// the "end" bit offset altogether.
	return uint32(want << start), mask
}
