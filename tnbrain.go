package main

import (
	"encoding/hex"
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

const ser = "/dev/ttyAMA0"

var (
	id   uint64 = 0
	port *serial.Port
)

// Here the channels are buffered, so that it is asynchronous
func SerialRead(out chan []byte) {
	var pkg []byte
	for {
		pkg = make([]byte, 27) // This is inefficient and reads extra
		_, err := port.Read(pkg)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("UART IN %s\n", pkg)
		out <- pkg
	}

}
func SerialWrite(in chan []byte) {
	var _msg, msg []byte
	for {
		_msg = <-in
		log.Printf("RAW: %x\n", _msg)
		msg = make([]byte, hex.EncodedLen(len(_msg)))
		hex.Encode(msg, _msg)
		msg = append([]byte("$"), msg...)
		msg = append(msg, []byte("\n")...)
		log.Printf("OUT: %s\n", msg)
		n, err := port.Write(msg)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Wrote %v bytes of data\n", n)
	}
}
func ToHavu(in, out chan string) {
	for {
		msg := <-in
		log.Printf("HAVU %s\n", msg)
		r := strings.NewReader(msg)
		_, err := http.Post("http://scout.polygame.org/api/msg", "text/plain", r)
		if err != nil {
			log.Println(err)
		}
	}
}

func MainLoop(in, out chan []byte, win, wout chan string) error {
	devs := make([]Relay, 1)
	devs[0] = Relay{"TB1", []string{"__:", "TB1"}, "", time.Now()}
	for {
		for _, dev := range devs {
			smac := tnparse.MACSuper{int(id), 1, 1, dev.path}
			hsmac := smac.NewMac()
			out <- hsmac

			submac := tnparse.MACSub{int(id), 0, tnparse.POSPOLL{}}
			out <- submac.NewSub()

			dat := <-in
			_smsg := dat[1 : len(dat)-1]
			smsg := make([]byte, hex.DecodedLen(len(_smsg)))
			hex.Decode(smsg, _smsg)

			mac := tnparse.MACSuper{}
			mac.FromTNH(smsg)

			for i := 0; i < mac.Pack_num; {
				dat := <-in
				_msg := dat[1 : len(dat)-1]
				msg := make([]byte, hex.DecodedLen(len(_msg)))
				hex.Decode(msg, _msg)
				mc := tnparse.MACSub{}
				mc.FromTNH(msg)
				switch t := mc.Packet.(type) {
				case tnparse.POSREPLY:
					p := mc.Packet.(tnparse.POSREPLY)
					switch _t := p.Pack.(type) {
					case tnparse.POS:
						_p := p.Pack.(tnparse.POS)
						wout <- _p.Havu
						i = mc.Pack_ord
					default:
						_ = _t
					}
				default:
					_ = t
				}
			}
			atomic.AddUint64(&id, 1)
		}
	}
}

func main() {
	conn := &serial.Config{Name: ser, Baud: 115200}
	_port, err := serial.OpenPort(conn)
	if err != nil {
		log.Fatal(err)
	}
	port = _port
	defer port.Close()
	log.Println("Hello")
	sin := make(chan []byte)
	sout := make(chan []byte)

	win := make(chan string, 50)
	wout := make(chan string, 50)
	go SerialRead(sin)
	go SerialWrite(sout)
	go ToHavu(wout, win)
	MainLoop(sin, sout, win, wout)
}
