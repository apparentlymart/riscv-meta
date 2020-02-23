package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func generateRustFragments(dir string, isa *ISA) error {
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}

	err = generateRustRawInstruction(filepath.Join(dir, "raw_instruction.rs"), isa.Arguments)
	err = generateRustInstruction(filepath.Join(dir, "instruction.rs"), isa)

	return nil
}

func generateRustRawInstruction(filename string, args map[string]*Argument) error {
	w, err := os.Create(filename)
	if err != nil {
		return err
	}

	w.WriteString("/// Represents a raw RISC-V instruction word that is yet to be decoded.\n")
	w.WriteString("///\n")
	w.WriteString("/// It can represent both standard-length and compressed instructions, the\n")
	w.WriteString("/// latter of which are supported by ignoring the higher-order parcel.\n")
	w.WriteString("pub struct RawInstruction (u32);\n")
	w.WriteString("\n")
	w.WriteString("impl RawInstruction {\n")
	w.WriteString("\n")

	// We'll include a method for each of the distinct argument types. It's
	// the responsibility of the caller to only call the methods appropriate
	// for a given instruction type, since otherwise the results will just
	// be garbage.

	var argNames []string
	for _, arg := range args {
		argNames = append(argNames, arg.Name)
	}
	sort.Strings(argNames)

	for _, name := range argNames {
		arg := args[name]
		resultTy := rustTypeForArgType(arg.Type, arg.EncWidth)
		fmt.Fprintf(w, "    pub fn %s(self) -> %s {\n", arg.FuncName, resultTy)
		if resultTy == "i32" {
			fmt.Fprintf(w, "        let width = %d;\n", arg.EncWidth)
		}
		if resultTy == "bool" && len(arg.Decoding) == 1 {
			// Simpler case for a single flag bit.
			fmt.Fprintf(w, "        return (self.0 & 0b%032b) != 0;\n", arg.Decoding[0].Mask)
		} else {
			w.WriteString("        let mut raw: u32 = 0;\n")
			for _, step := range arg.Decoding {
				switch {
				case step.RightShift == 0:
					fmt.Fprintf(w, "        // Fill 0b%032b\n", step.Mask)
					fmt.Fprintf(w, "        raw |= (self.0 & 0b%032b);\n", step.Mask)
				case step.RightShift < 0:
					fmt.Fprintf(w, "        // Fill 0b%032b\n", step.Mask<<-step.RightShift)
					fmt.Fprintf(w, "        raw |= (self.0 & 0b%032b) << %d;\n", step.Mask, -step.RightShift)
				default:
					fmt.Fprintf(w, "        // Fill 0b%032b\n", step.Mask>>step.RightShift)
					fmt.Fprintf(w, "        raw |= (self.0 & 0b%032b) >> %d;\n", step.Mask, step.RightShift)
				}
			}
			switch resultTy {

			case "u32":
				w.WriteString("        return raw;\n")
			case "i32":
				w.WriteString("        return sign_extend(raw, width);\n")
			case "IntRegister":
				w.WriteString("        return IntRegister::num(raw as usize);\n")
			case "FloatRegister":
				w.WriteString("        return FloatRegister::num(raw as usize);\n")
			default:
				fmt.Fprintf(w, "        // ERROR: don't know how to build %s result\n", resultTy)
			}
		}
		w.WriteString("    }\n")
		w.WriteString("\n")
	}

	w.WriteString("}\n")
	return nil
}

func generateRustInstruction(filename string, isa *ISA) error {
	w, err := os.Create(filename)
	if err != nil {
		return err
	}

	for _, isaSize := range []Size{RV32, RV64, RV128} {
		w.WriteString("\n")
		fmt.Fprintf(w, "/// Enumeration of all operations from the RV%d ISA.\n", int(isaSize))
		fmt.Fprintf(w, "enum OperationRV%d {\n", int(isaSize))

		for _, ext := range []Extension{ExtI, ExtM, ExtA, ExtS, ExtF, ExtD, ExtQ, ExtC} {
			extName := isa.ExtensionNames[ext]
			fmt.Fprintf(w, "\n    // RV%d%c: %s\n\n", int(isaSize), byte(ext), extName)

			std := MakeStandard(isaSize, ext)

			for _, op := range isa.Ops {
				if !op.Standards.Has(std) {
					continue
				}
				fmt.Fprintf(w, "    /// %s (RV%d%c)\n", op.FullName, int(isaSize), byte(ext))
				if len(op.Codec.Operands) == 0 {
					fmt.Fprintf(w, "    %s;\n", op.TypeName)
					continue
				}
				fmt.Fprintf(w, "    %s {\n", op.TypeName)
				for _, argName := range op.Codec.Operands {
					arg := isa.Arguments[argName]
					rustType := rustTypeForArgType(arg.Type, arg.EncWidth)
					fmt.Fprintf(w, "        %s: %s;\n", arg.FuncLocalName, rustType)
				}
				w.WriteString("    }\n")
			}
		}

		w.WriteString("\n}\n")
	}

	return nil
}

func rustTypeForArgType(ty ArgType, encWidth int) string {
	switch ty {
	case ArgIntReg, ArgCompressedReg:
		return "IntRegister"
	case ArgFloatReg:
		return "FloatRegister"
	case ArgOffset, ArgSignedImmediate:
		return "i32"
	default:
		if encWidth == 1 {
			return "bool"
		}
		return "u32"
	}
}