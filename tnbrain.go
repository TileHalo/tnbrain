package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/TileHalo/tnparse"
	"github.com/tarm/serial"
)

type (
	Relay struct {
		id     string
		path   []string
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

const port = "/dev/ttyAMA0"

var id uint64 = 0

// Here the channels are buffered, so that it is asynchronous
func Serial(in, out chan []byte) {
	conn := &serial.Config{Name: "TTYAMA0", Baud: 115200}
	port, err := serial.OpenPort(conn)
	if err != nil {
		log.Fatal(err)
	}
	defer port.Close()

	var msg []byte
	var pkg []byte
	for {
		pkg = make([]byte, 27) // This is inefficient and reads extra
		msg = <-in
		_, err = port.Read(pkg)
		if err != nil {
			log.Fatal(err)
		}
		_, nerr := port.Write(msg)
		if nerr != nil {
			log.Fatal(err)
		}
		out <- pkg
	}

}
func ToHavu(in, out chan string) {
	for {
		msg := <-in
		r := strings.NewReader(msg)
		_, err := http.Post("http://scout.polygame.org/api/msg", "text/plain", r)
		if err != nil {
			log.Println(err)
		}
	}
}

func MainLoop(in, out chan []byte, win, wout chan string) error {
	devs := make([]Relay, 1)
	devs[1] = Relay{"TB1", []string{""}, "", time.Now()}
	for {
		for _, dev := range devs {
			smac := tnparse.MACSuper{int(id), 1, 0, dev.path}
			out <- smac.NewMac()
			cont := true
			for cont {

			}
			fmt.Println(dev.id)

			atomic.AddUint64(&id, 1)
		}
	}
}

func main() {
	sin := make(chan []byte, 50)
	sout := make(chan []byte, 50)

	win := make(chan string, 50)
	wout := make(chan string, 50)
	go Serial(sout, sin)
	go ToHavu(wout, win)
	MainLoop(sin, sout, win, wout)
}
