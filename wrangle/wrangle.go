package main

import (
	"log"

	"github.com/davecgh/go-spew/spew"
)

func main() {
	isa, err := loadISAMeta()
	if err != nil {
		log.Fatal(err)
	}

	spew.Dump(isa)
	generateRustFragments("generated/rust", isa)
}
