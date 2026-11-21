package sim

import (
	"bufio"
	"encoding/json"
	"io"
	"os/exec"

	"mechbot/proto"
)

// ExtBot drives a mech with an external process speaking mechfight/1 on
// stdin/stdout (a built bot binary), so versions can be played against each other.
type ExtBot struct {
	cmd *exec.Cmd
	in  io.WriteCloser
	out *bufio.Reader
	bad bool
}

func NewExtBot(path string, h *proto.Hello) *ExtBot {
	e := &ExtBot{}
	e.cmd = exec.Command(path)
	var err error
	if e.in, err = e.cmd.StdinPipe(); err != nil {
		e.bad = true
		return e
	}
	so, err := e.cmd.StdoutPipe()
	if err != nil {
		e.bad = true
		return e
	}
	e.out = bufio.NewReaderSize(so, 1<<16)
	if err := e.cmd.Start(); err != nil {
		e.bad = true
		return e
	}
	hb, _ := json.Marshal(h)
	e.in.Write(append(hb, '\n'))
	if _, err := e.out.ReadBytes('\n'); err != nil {
		e.bad = true
	}
	return e
}

func (e *ExtBot) Act(o *proto.Obs) proto.Action {
	if e.bad {
		return proto.Action{Tick: -1}
	}
	b, _ := json.Marshal(o)
	if _, err := e.in.Write(append(b, '\n')); err != nil {
		e.bad = true
		return proto.Action{Tick: -1}
	}
	ln, err := e.out.ReadBytes('\n')
	if err != nil {
		e.bad = true
		return proto.Action{Tick: -1}
	}
	var a proto.Action
	if json.Unmarshal(ln, &a) != nil {
		return proto.Action{Tick: -1}
	}
	return a
}

func (e *ExtBot) Close() {
	if e.in != nil {
		e.in.Write([]byte(`{"type":"end","place":1,"kills":0}` + "\n"))
		e.in.Close()
	}
	if e.cmd != nil && e.cmd.Process != nil {
		e.cmd.Wait()
	}
}
