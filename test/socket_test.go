package test

import (
	"fmt"
	"gsocket"
	"testing"
)

func TestRegister(t *testing.T) {
	gsocket.RegisterRun()
	fmt.Println("test vscode git")
}
func TestGateway(t *testing.T) {
	gsocket.GatewayRun()
}
func TestWorker(t *testing.T) {
	gsocket.WorkerRun()
}
