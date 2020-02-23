package main

import (
	"fmt"
	"strconv"
	"strings"
)

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

func rangeMask(top, bottom uint) bits32 {
	return bits32((1 << (top + 1)) - (1 << bottom))
}

type ArgDecodeStep struct {
	Mask       bits32
	RightShift int
}

func (s ArgDecodeStep) String() string {
	switch {
	case s.RightShift == 0:
		return fmt.Sprintf("(inst & %s)", s.Mask.String())
	case s.RightShift < 0:
		return fmt.Sprintf("(inst & %s) << %d", s.Mask.String(), -s.RightShift)
	default:
		return fmt.Sprintf("(inst & %s) >> %d", s.Mask.String(), s.RightShift)
	}
}

func ParseArgDecodeSteps(raw string) []ArgDecodeStep {
	// Deals with strings like these from the "operands" file and normalizes
	// them to just be a sequence of "mask, then shift" operations whose
	// results can be bitewise-ORed together to produce the final value.

	parts := strings.Split(raw, ",")
	var ret []ArgDecodeStep
	for _, rawPart := range parts {
		brack := strings.IndexByte(rawPart, '[')
		switch {
		case brack == -1:
			// A simple left-justified field, then.
			rawTop, rawBottom := partition(rawPart, ":")
			if rawBottom == "" {
				rawBottom = rawTop
			}
			top, err := strconv.ParseUint(rawTop, 10, 64)
			if err != nil {
				continue
			}
			bottom, err := strconv.ParseUint(rawBottom, 10, 64)
			if err != nil {
				continue
			}
			mask := rangeMask(uint(top), uint(bottom))

			ret = append(ret, ArgDecodeStep{
				Mask:       mask,
				RightShift: int(bottom),
			})

		default:
			// A more complicated sequence of operations gathering values
			// for a single field from several separate sources. In this
			// case we might generate multiple decode steps because a
			// consecutive sequence of bits in the input can become
			// non-consecutive in the output.
			rawSrc, rawDests := partition(rawPart, "[")
			rawDests = rawDests[:len(rawDests)-1] // trim closing bracket

			rawSrcTop, _ := partition(rawSrc, ":")
			srcTop, err := strconv.ParseUint(rawSrcTop, 10, 64)
			if err != nil {
				continue
			}

			rawConcats := strings.Split(rawDests, "|")
			for _, rawConcat := range rawConcats {
				rawDestTop, rawDestBottom := partition(rawConcat, ":")
				if rawDestBottom == "" {
					rawDestBottom = rawDestTop
				}
				destTop, err := strconv.ParseUint(rawDestTop, 10, 64)
				if err != nil {
					continue
				}
				destBottom, err := strconv.ParseUint(rawDestBottom, 10, 64)
				if err != nil {
					continue
				}
				width := destTop - destBottom
				srcBottom := srcTop - width

				mask := rangeMask(uint(srcTop), uint(srcBottom))
				ret = append(ret, ArgDecodeStep{
					Mask:       mask,
					RightShift: int(srcBottom) - int(destTop),
				})

				// The next concat will pick up where this one left off, so
				// we'll push srcTop along by the width of what we just decoded.
				srcTop -= width + 1
			}

		}
	}
	return ret
}
