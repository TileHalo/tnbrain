package main

import (
	"encoding/hex"
	"log"
	"time"

	"github.com/tarm/serial"
)

type (
	Relay struct {
		id     string
		links  string
		status string
		last   time.Time
	}

	Device struct {
		id   string
		last time.Time
		lmsg *DevMsg
	}

	DevMsg struct {
		id       string
		heard_by string
		stamp    time.Time
		msg      string
	}
)

func StrToBinary(in string) string {
	out := make(string)
	for _, c := range in {
		out = fmt.Sprintf(":s:.8b", out, c)
	}
	return out
}

func StrToHex(in string) string {
	src := []byte(in)
	dst := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(dst, src)
	return string(dst)
}

func HexToStr(src []byte) (string, error) {
	dst := make([]byte, hex.DecodedLen(len(src)))
	_, err := hex.Decode(dst, src)
	return string(dst), err
}

// Here the channels are buffered, so that it is asynchronous
func Serial(in, out chan string) {
	conn := &serial.Config{Name: "TTYAMA0", Baud: 115200}
	port, err := serial.OpenPort(conn)
	if err != nil {
		log.Fatal(err)
	}
	defer port.Close()

	var msg string
	var pkg []byte
	for {
		pkg = make([]byte, 27) // This is inefficient and reads extra
		msg = <-in
		_, err = port.Read(pkg)
		if err != nil {
			log.Fatal(err)
		}
		out <- StrToBinary(HexToStr(pkg))
	}

}

func WS(in, out chan string) {

}

func MainLoop(in, out chan string) error {
}

func main() {
	sin := make(chan string, 50)
	sout := make(chan string, 50)

	win := make(chan string, 50)
	wout := make(chan string, 50)
	go Serial(sout, sin)
}
