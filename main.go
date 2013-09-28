package main

import "bufio"
import "fmt"
import "io"
import "net"
import "net/url"
import "os"
import "regexp"
import "time"

func main() {
	port := 1500
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	checkError(err)

	fmt.Printf("Listening at :%d.\n", port)
	for {
		listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))

		conn, err := listener.Accept()
		if err != nil {
			continue;
		}

		 go handleConnection(conn)
	}
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var queryPattern = regexp.MustCompile("^([A-Z]+) (.*?) HTTP")
var portPattern = regexp.MustCompile(":\\d{1,5}$")

// Handles a single client connection.
func handleConnection(conn net.Conn) (err error) {
	defer conn.Close()

	timestamp := time.Now().UnixNano()

	conn.SetDeadline(time.Now().Add(100 * time.Millisecond))

	in := bufio.NewReader(conn)

	buffer, err := in.Peek(1024)
	query := queryPattern.FindStringSubmatch(string(buffer))
	if query == nil {
		err = fmt.Errorf("Failed to parse HTTP query header.")
		return
	}

	uri, err := url.ParseRequestURI(query[2])
	if err != nil {
		return
	}

	var target string
	if portPattern.MatchString(uri.Host) {
		target = uri.Host
	} else {
		target = fmt.Sprintf("%s:80", uri.Host)
	}


	fmt.Printf("Handling request for %s.\n", uri)

	host, err := net.Dial("tcp", target)
	if err != nil {
		return
	}
	defer host.Close()

	reqLog, err := os.Create(fmt.Sprintf("%d_req", timestamp))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	resLog, err := os.Create(fmt.Sprintf("%d_res", timestamp))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	err = pipe(in, tee { host, reqLog, 1000 }, 100)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	err = pipe(host, tee { conn, resLog, 1000 }, 2000)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	return
}

// Pipes the input stream to the output stream.
func pipe(in io.Reader, out io.Writer, timeout int) (err error) {
	buffer := make([]byte, 1024)
	for {
		if conn, ok := in.(net.Conn); ok {
			conn.SetDeadline(time.Now().Add(time.Duration(timeout) * time.Millisecond))
		}

		inCount, err1 := in.Read(buffer)
		if err1 != nil && !err1.(net.Error).Timeout() {
			err = err1
			return
		}

		n := 0
		for n < inCount {
			if conn, ok := out.(net.Conn); ok {
				conn.SetDeadline(time.Now().Add(time.Duration(timeout) * time.Millisecond))
			}

			outCount, err2 := out.Write(buffer[n:inCount])
			if err2 != nil && !err2.(net.Error).Timeout() {
				err = err2
				return
			}

			n += outCount
		}

		if err1 != nil || inCount == 0 {
			break
		}
	}

	return
}
