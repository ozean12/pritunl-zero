package main

import (
	"github.com/ozean12/pritunl-zero/cmd"
	"github.com/ozean12/pritunl-zero/constants"
	"testing"
)

func TestServer(t *testing.T) {
	constants.Production = false

	Init()
	err := cmd.Node(true)
	if err != nil {
		panic(err)
	}

	return
}
