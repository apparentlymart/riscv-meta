package main

import (
	"fmt"
)

type bits8 uint8
type bits32 uint32

func (v bits8) String() string {
	return fmt.Sprintf("0b%08b", v)
}

func (v bits32) String() string {
	return fmt.Sprintf("0b%032b", v)
}
