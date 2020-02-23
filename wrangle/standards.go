package main

import (
	"fmt"
	"sort"
	"strings"
)

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
