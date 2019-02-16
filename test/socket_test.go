package test

import (
	"gsocket"
	"testing"
)

func TestRegister(t *testing.T) {
	gsocket.RegisterRun()
}
func TestGateway(t *testing.T) {
	gsocket.GatewayRun()
}
func TestWorker(t *testing.T) {
	gsocket.WorkerRun()
}
