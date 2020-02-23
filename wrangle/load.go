package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

func loadISAMeta() (*ISA, error) {
	extNames, err := loadExtensionNames("extensions")
	if err != nil {
		return nil, fmt.Errorf("failed to load extension names: %s", err)
	}
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
		ExtensionNames: extNames,
		MajorOpcodes:   majorOpcodes,
		Codecs:         codecs,
		Arguments:      args,
		Ops:            ops,
		Expansions:     exps,
	}, nil
}

func loadExtensionNames(filename string) (map[Extension]string, error) {
	r, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	ret := make(map[Extension]string)

	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := trimComments(sc.Text())
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		if fields[1] != "32" {
			// We're cheating here and using the RV32 names only because we
			// want to use a single name for each extension letter, ignoring
			// the separate 64-bit and 128-bit variants. (The larger variants
			// are usually just the same text with "in addition to RV32..."
			// appended anyway.)
			continue
		}

		ext := Extension(strings.ToUpper(fields[2])[0])

		quot := strings.IndexRune(line, '"')
		if quot < 0 {
			continue
		}
		name := line[quot+1:]
		quot = strings.IndexRune(name, '"')
		if quot >= 0 {
			name = name[:quot]
		}

		// Trim off "RV32x " prefix, because we're using the 32-bit form's
		// name for all of them.
		name = name[6:]

		// The "Standard Extension For" prefix is also redundant, so we'll
		// trim it to make these things more compact.
		const stdExtFor = "Standard Extension for "
		if strings.HasPrefix(name, stdExtFor) {
			name = name[len(stdExtFor):]
		}

		ret[ext] = strings.TrimSpace(name)
	}

	return ret, sc.Err()
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

		decoding, encWidth := ParseArgDecodeSteps(fields[1])

		arg := &Argument{
			Name:          name,
			FuncName:      makeIdentUnderscores(name),
			TypeName:      makeIdentTitle(name),
			FuncLocalName: strings.ReplaceAll(makeIdentUnderscores(fields[3]), "_", ""),
			TypeLocalName: makeIdentTitle(fields[3]),
			Type:          ArgType(fields[2]),
			EncWidth:      encWidth,
			Decoding:      decoding,
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

func trimComments(line string) string {
	hash := strings.IndexByte(line, '#')
	if hash == -1 {
		return line
	}
	return line[:hash]
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
	mask = uint32(rangeMask(uint(end), uint(start)))

	// We're just assuming that there won't be a "val" that is too
	// big to fit in the identified bits here, which means we can ignore
	// the "end" bit offset altogether.
	return uint32(want << start), mask
}
