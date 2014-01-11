package replica

import (
	"fmt"
	"testing"

	cmd "github.com/go-epaxos/epaxos/command"
)

var _ = fmt.Printf

func TestRecvPropose(t *testing.T) {
	r := startNewReplica(0, 5)
	messageChan := make(chan Message)
	propose := &Propose{
		Cmds: []cmd.Command{
			cmd.Command("hello"),
			cmd.Command("world"),
		},
	}
	r.recvPropose(propose, messageChan)

	for i := 0; i < r.N-1; i++ {
		message := <-messageChan
		preAccept := message.(*PreAccept)

		if preAccept.Cmds[0].Compare(propose.Cmds[0]) != 0 ||
			preAccept.Cmds[1].Compare(propose.Cmds[1]) != 0 {
			t.Fatal("command isn't equal")
		}
	}

	// check deps, seq and instance id
	propose = &Propose{
		Cmds: []cmd.Command{
			cmd.Command("hello"),
			cmd.Command("world"),
		},
	}
	r.recvPropose(propose, messageChan)

	for i := 0; i < r.N-1; i++ {
		message := <-messageChan
		preAccept := message.(*PreAccept)

		if preAccept.Cmds[0].Compare(propose.Cmds[0]) != 0 ||
			preAccept.Cmds[1].Compare(propose.Cmds[1]) != 0 {
			t.Fatal("command isn't equal")
		}
		if preAccept.InsId != 1 {
			t.Fatal("instance id should be 1")
		}
		if preAccept.Seq != 1 {
			t.Fatal("seq should be 1")
		}
		if preAccept.Deps[0] != 0 {
			t.Fatal("deps[0] should be 0")
		}
	}
}
