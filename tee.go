package main

import "io"
import "net"
import "time"

// Writes a stream of data to two output streams.
type tee struct {
	a io.Writer
	b io.Writer
	timeout int
}

func (t tee) Write(buffer []byte) (n int, err error) {
	if conn, ok := t.a.(net.Conn); ok {
		conn.SetDeadline(time.Now().Add(time.Duration(t.timeout) * time.Millisecond))
	}

	n, err = t.a.Write(buffer)
	if err != nil {
		return
	}

	if conn, ok := t.b.(net.Conn); ok {
		conn.SetDeadline(time.Now().Add(time.Duration(t.timeout) * time.Millisecond))
	}

	n, err = t.b.Write(buffer[:n])

	return
}
