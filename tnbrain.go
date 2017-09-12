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
	pack_time        = 60 // ms
	id        uint64 = 0
	port      *serial.Port
	logfile   = "tacnetlog.log"
)

func CreateMacs(dev Relay, pkg tnparse.TNH) (tnparse.MACSuper, tnparse.MACSub) {
	smac := tnparse.MACSuper{int(id), 1, 1, dev.path}

	submac := tnparse.MACSub{int(id), 0, pkg}
	return smac, submac
}

func Timeout(t chan bool, dur int) {
	time.Sleep(time.Duration(dur) * time.Second)
	t <- true
}

// Here the channels are buffered, so that it is asynchronous
func SerialRead(out chan []byte) {
	msgs := []byte{}
	reading := false
	for {
		msg := make([]byte, 1)
		_, err := port.Read(msg)
		if err != nil {
			log.Fatal("READ")
		}
		if msg[0] == '%' || msg[0] == '$' {
			reading = true
		} else if msg[0] == '\n' && reading == true {
			reading = false
			_msg := make([]byte, hex.DecodedLen(len(msgs)))
			hex.Decode(_msg, msgs)
			out <- _msg
			log.Printf("READ: %x\n", _msg)
			msgs = []byte{}
		} else if reading == true {
			msgs = append(msgs, msg...)
		}
	}
}
func SerialWrite(in chan []byte) {
	var _msg, msg []byte
	for {
		_msg = <-in
		msg = make([]byte, hex.EncodedLen(len(_msg)))
		hex.Encode(msg, _msg)
		msg = append([]byte("$"), msg...)
		msg = append(msg, []byte("\n")...)
		_, err := port.Write(msg)
		if err != nil {
			log.Fatal(err)
		}
	}
}
func ToHavu(in, out chan string) {
	for {
		msg := <-in
		log.Printf("HAVU %s\n", msg)
		r := strings.NewReader(msg)
		_, err := http.Post("http://hylly.havu.org/api/msg", "text/plain", r)
		if err != nil {
			log.Println(err)
		}
	}
}

func MainLoop(in, out chan []byte, win, wout chan string) error {
	devs := []Relay{}
	devs = append(devs, Relay{"KB1", []string{"__:", "KB1"}, "", time.Now()})
	devs = append(devs, Relay{"TB1", []string{"__:", "KB1", "TB1"}, "", time.Now()})
	// devs = append(Relay{"TB2", []string{"__:", "KB1", "TB2"}, "", time.Now()})
	for {
		for _, dev := range devs {
			tt := make(chan bool)
			log.Printf("Polling device %s\n", dev.id)
			smac, submac := CreateMacs(dev, &tnparse.POSPOLL{})
			out <- smac.ToTNH()
			out <- submac.ToTNH()

			var smsg []byte
			// tout := float64(pack_time*(len(dev.path)-2))/1000.0 + 0.4
			go Timeout(tt, 5)
			select {
			case smsg = <-in:

				mac := tnparse.MACSuper{}
				mac = mac.FromTNH(smsg).(tnparse.MACSuper)
				tt = make(chan bool)
				for i := 0; i < mac.Pack_num; {
					var msg []byte
					go Timeout(tt, 5.0)
					select {
					case msg = <-in:
						mc := tnparse.MACSub{}
						mc = mc.FromTNH(msg).(tnparse.MACSub)
						switch mc.Packet.(type) {
						case tnparse.POSREPLY:
							p := mc.Packet.(tnparse.POSREPLY)
							switch p.Pack.(type) {
							case tnparse.POS:
								_p := p.Pack.(tnparse.POS)
								_p = _p.CalculateHavu(dev.id)
								wout <- _p.Havu
								i = mc.Pack_ord
							}
						default:
							i = mac.Pack_num
						}
					case <-tt:
						log.Printf("TIMEOUT RECEIVING PACKETS %s\n", dev.id)
						i = mac.Pack_num
					}
				}
				atomic.AddUint64(&id, 1)
			case <-tt:
				log.Printf("TIMEOUT CALLING %s\n", dev.id)
			}
		}
	}
}

func main() {

	// Enable this to log into a file
	// f, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	// if err != nil {
	// 	fmt.Printf("error opening file: %v", err)
	// }
	// log.SetOutput(f)
	// defer f.Close()

	conn := &serial.Config{Name: ser, Baud: 115200}
	_port, err := serial.OpenPort(conn)
	if err != nil {
		log.Fatal(err)
	}
	port = _port
	defer port.Close()

	log.Println("TACNET starting")

	sin := make(chan []byte)
	sout := make(chan []byte)

	win := make(chan string, 50)
	wout := make(chan string, 50)
	go SerialRead(sin)
	go SerialWrite(sout)
	go ToHavu(wout, win)
	MainLoop(sin, sout, win, wout)
}
