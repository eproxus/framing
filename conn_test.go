package framing_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/eproxus/framing"
)

func Test1BigEndian(t *testing.T) {
	run(t, 1, binary.BigEndian)
}

func Test2BigEndian(t *testing.T) {
	run(t, 2, binary.BigEndian)
}

func Test4BigEndian(t *testing.T) {
	run(t, 4, binary.BigEndian)
}

func Test1LittleEndian(t *testing.T) {
	run(t, 1, binary.LittleEndian)
}

func Test2LittleEndian(t *testing.T) {
	run(t, 2, binary.LittleEndian)
}

func Test4LittleEndian(t *testing.T) {
	run(t, 4, binary.LittleEndian)
}

func run(t *testing.T, size byte, endianess binary.ByteOrder) {
	message := "13 bytes long"

	l, err := net.Listen("tcp", ":0") // listen on localhost
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	port := l.Addr().(*net.TCPAddr).Port

	go func() {
		conn, err := net.Dial("tcp", ":"+strconv.Itoa(port))
		if err != nil {
			t.Fatal(err)
		}

		framed, err := framing.NewConn(conn, size, endianess)
		if err != nil {
			t.Fatal(err)
		}
		defer framed.Close()

		for i := 1; i <= 2; i++ {
			if _, err := fmt.Fprintf(framed, message); err != nil {
				t.Fatal(err)
			}
		}
	}()

	conn, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}
	framed, err := framing.NewConn(conn, size, endianess)
	if err != nil {
		t.Fatal(err)
	}
	defer framed.Close()

	buf, err := framed.ReadFrame()
	if err != nil {
		t.Fatal(err)
	}
	if msg := string(buf[:]); msg != message {
		t.Fatalf("Unexpected message:\nGot:\t\t%s\nExpected:\t%s\n", msg, message)
	}

	fixed := make([]byte, 20) // More than 13
	n, err := framed.Read(fixed)
	if err != nil {
		t.Fatal(err)
	}
	if n != 13 {
		t.Fatal("Frame is not of correct size")
	}
	if msg := string(fixed[:13]); msg != message {
		t.Fatalf("Unexpected message:\nGot:\t\t%s\nExpected:\t%s\n", msg, message)
	}

	return // Done
}

func TestInvalidFrameSiz(t *testing.T) {
	_, err := framing.NewConn(nil, 3, nil)
	if err != framing.ErrPrefixLength {
		t.Fail()
	}
}

func TestPacketTooLarge1(t *testing.T) {
	packetTooLarge(t, 1, math.MaxUint8)
}

func TestPacketTooLarge2(t *testing.T) {
	packetTooLarge(t, 2, math.MaxUint16)
}

func TestPacketTooLarge4(t *testing.T) {
	packetTooLarge(t, 4, math.MaxUint32)
}

func packetTooLarge(t *testing.T, size byte, max uint) {
	conn, err := framing.NewConn(nil, size, nil)
	if err != nil {
		t.Fatal(err)
	}
	b := make([]byte, max+1)
	_, err = conn.Write(b)
	if err != framing.ErrFrameTooLarge {
		t.Fail()
	}
}

func TestBufferTooSmall(t *testing.T) {
	conn := &fake{ReadWriter: bytes.NewBuffer([]byte{4, 0, 0, 0, 0})}
	framed, _ := framing.NewConn(conn, 1, nil)

	b := make([]byte, 2)
	n, err := framed.Read(b)
	if err != framing.ErrBufferTooSmall {
		t.Fatal("No error received, actual bytes:", n)
	}
}

func TestInnerReadSizeError(t *testing.T) {
	conn := &fake{
		ReadWriter: bytes.NewBuffer([]byte{}),
		err:        errors.New("InnerReadSizeError"),
	}
	framed, _ := framing.NewConn(conn, 1, nil)
	b := make([]byte, 4)
	_, err := framed.Read(b)
	if err.Error() != "InnerReadSizeError" {
		t.Fatal(err)
	}
}

func TestInnerReadError(t *testing.T) {
	conn := &fake{
		ReadWriter: bytes.NewBuffer([]byte{4}),
		err:        errors.New("InnerReadError"),
	}
	framed, _ := framing.NewConn(conn, 1, nil)
	b := make([]byte, 4)
	_, err := framed.Read(b)
	if err.Error() != "InnerReadError" {
		t.Fatal(err)
	}
}

func TestInnerReadFrameSizeError(t *testing.T) {
	conn := &fake{
		ReadWriter: bytes.NewBuffer([]byte{}),
		err:        errors.New("InnerReadFrameSizeError"),
	}
	framed, _ := framing.NewConn(conn, 1, nil)
	_, err := framed.ReadFrame()
	if err.Error() != "InnerReadFrameSizeError" {
		t.Fatal(err)
	}
}

func TestInnerReadFrameError(t *testing.T) {
	conn := &fake{
		ReadWriter: bytes.NewBuffer([]byte{4}),
		err:        errors.New("InnerReadFrameError"),
	}
	framed, _ := framing.NewConn(conn, 1, nil)
	_, err := framed.ReadFrame()
	if err.Error() != "InnerReadFrameError" {
		t.Fatal(err)
	}
}

func TestWriteError(t *testing.T) {
	conn := &fake{
		ReadWriter: bytes.NewBuffer([]byte{}),
		err:        errors.New("WriteError"),
	}
	framed, _ := framing.NewConn(conn, 1, nil)
	_, err := framed.Write([]byte{0})
	if err.Error() != "WriteError" {
		t.Fatal(err)
	}
}

type fake struct {
	io.ReadWriter
	err error
}

func (f *fake) Read(b []byte) (int, error) {
	n, err := f.ReadWriter.Read(b)
	if n == 0 {
		if f.err != nil {
			return 0, f.err
		}
		return 0, err
	}
	return n, err
}

func (f *fake) Write(b []byte) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.ReadWriter.Write(b)
}

func (f *fake) Close() error {
	return nil
}

func (f *fake) LocalAddr() net.Addr {
	return nil
}

func (f *fake) RemoteAddr() net.Addr {
	return nil
}

func (f *fake) SetDeadline(time time.Time) error {
	return nil
}

func (f *fake) SetReadDeadline(time time.Time) error {
	return nil
}

func (f *fake) SetWriteDeadline(time time.Time) error {
	return nil
}
