package test

import (
	"testing"
	"gsocket/client"
)

func TestClientA(t *testing.T) {
	client.ClientASend()
}
func TestClientB(t *testing.T) {
	client.ClientBRecive()
}
