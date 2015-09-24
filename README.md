[![Build Status](https://travis-ci.org/eproxus/framing.svg)](https://travis-ci.org/eproxus/framing)
[![Coverage Status](https://coveralls.io/repos/eproxus/framing/badge.svg?branch=master&service=github)](https://coveralls.io/github/eproxus/framing?branch=master)
[![GoDoc](https://godoc.org/github.com/eproxus/framing?status.svg)](https://godoc.org/github.com/eproxus/framing)
![Go Version](https://img.shields.io/badge/go-1.5-5272B4.svg)

# framing
Framing provides a prefix length framed net.Conn connection. This is useful if
you have connections that send packages that are prefixed with 1, 2 or 4 bytes
of message length before the actual data.

## Example

```go
package main

import (
    "encoding/binary"
    "fmt"
    "log"
    "net"
    "strconv"

    "github.com/eproxus/framing"
)

const prefixLength = 4

var endianess = binary.BigEndian

func main() {
    message := "13 bytes long"

    l, err := net.Listen("tcp", ":0") // Listen on localhost, random port
    if err != nil {
        log.Fatal(err)
    }
    defer l.Close()

    port := l.Addr().(*net.TCPAddr).Port

    // Send message in a go routine
    go func() {
        conn, err := net.Dial("tcp", ":"+strconv.Itoa(port))
        if err != nil {
            log.Fatal(err)
        }

        framed, err := framing.NewConn(conn, prefixLength, endianess)
        if err != nil {
            log.Fatal(err)
        }
        defer framed.Close()

        if _, err := fmt.Fprintf(framed, message); err != nil {
            log.Fatal(err)
        }
    }()

    conn, err := l.Accept()
    if err != nil {
        log.Fatal(err)
    }

    framed, err := framing.NewConn(conn, prefixLength, endianess)
    if err != nil {
        log.Fatal(err)
    }
    defer framed.Close()

    // Receive message
    frame, err := framed.ReadFrame()
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Recieved \"%v\"\n", string(frame[:13]))
}
```
