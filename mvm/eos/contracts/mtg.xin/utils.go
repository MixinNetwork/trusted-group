
package main

import (
	"github.com/uuosio/chain"
)

func check(b bool, msg string) {
	chain.Check(b, msg)
}
