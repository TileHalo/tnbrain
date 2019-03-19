package main

import (
	"encoding/hex"
	"fmt"
	"github.com/tarm/serial"
	// "log"
	"net/http"
	"os"
)

const ser = "/dev/ttyUSB1"

var (
	pack_time        = 60 // ms
	id        uint64 = 0
	// logfile          = "tacnetlog.log"
	// posfile          = "pos.log"
	// disfile          = "discard.log"
	myid = "KC1"
	// posfd     *os.File
	// disfd     *os.File
	port *serial.Port
)

type POS struct {
	Havu     string
	sender   string
	received uint
	northing uint
	easting  uint
	height   float64
	speed    float64
	bearing  int
	voltage  int
	temp     int
	distress int
	status   int
	angle    int
	rssi     int
}

func (p POS) FromTNH(message []byte) POS {

	// RSSI is sent as the first byte
	rssibyte := message[0]
	for i := 0; i < 15; i++ {
		message[i] = message[i+1]
	}

	sendbyte := make([]byte, 3)
	sendbyte[0] = ((message[0] & 15) << 1) + ((message[1] & 128) >> 7) + 65
	sendbyte[1] = ((message[1] & 124) >> 2) + 65
	sendbyte[2] = ((message[1] & 3) << 2) + ((message[2] & 192) >> 6) + 48
	p.sender = fmt.Sprintf("%s", sendbyte)

	var t uint
	t = uint((message[2] & 63)) << 11
	t += (uint(message[3]) << 3)
	t += uint(message[4]&224) >> 5
	p.received = t

	northing := uint(0)
	northing += uint((message[4] & 31)) << 16
	northing += uint((message[5])) << 8
	northing += uint(message[6])
	northing += 6000000
	p.northing = northing

	easting := uint32(0)
	easting += uint32(message[7]) << 12
	easting += uint32(message[8]) << 4
	easting += uint32((message[9] & 240)) >> 4
	p.easting = uint(easting)

	p.height = float64(((message[9]&15)<<7)+((message[10]&254)>>1)) * 2
	p.speed = float64((uint(message[10]&1)<<8)+(message[11])) / 10
	p.angle = int((message[12]&252)>>2) * 6
	p.voltage = 5000 + 10*int(((message[12]&3)<<6)+((message[13]&252)>>2))
	temp := ((message[13] & 3) << 4) + ((message[14] & 240) >> 4)
	if temp > 32 {
		p.temp = -int(((temp ^ 0xff) + 1) & 31)
	} else {
		p.temp = int(temp & 31)
	}
	p.status = int((message[14] & 14) >> 1)
	p.distress = int(message[14] & 1)
	p.rssi = -1 * int(rssibyte)

	var h, m, s int
	h = int(t) / 3600
	m = (int(t) - h*3600) / 60
	s = int(t) - h*3600 - m*60
	p.Havu = fmt.Sprintf("$POS|ETRS-TM35FIN|%s||%d:%d:%d|%d|%d|%.2f|%d*",
		p.sender,
		h,
		m,
		s,
		p.northing,
		p.easting,
		p.height,
		p.distress)
	return p
}
func (p POS) CalculateHavu() POS {

	t := p.received
	var h, m, s int
	h = int(t) / 3600
	m = (int(t) - h*3600) / 60
	s = int(t) - h*3600 - m*60
	p.Havu = fmt.Sprintf("$TACPOS|ETRS-TM35FIN|%s||UTC%d:%d:%d|%d|%d|%.2f|||OK|%d||%d|%s|%d|||*",
		p.sender,
		h,
		m,
		s,
		p.northing,
		p.easting,
		p.height,
		p.distress,
		p.voltage,
		myid,
		p.rssi)
	return p
}

func (p POS) ToTNH() []byte {
	return make([]byte, 17)
}

func ToHavu(in, out chan string) {
	get_fmt := "http://scout.polygame.fi/api/msg?msg=%s"
	for {
		msg := <-in
		// log.Printf("HAVU %s\n", msg)
		resp, err := http.Get(fmt.Sprintf(get_fmt, msg))
		if resp.StatusCode != 200 || err != nil {
			if err != nil {
				// log.Println(err)
			}
			// log.Printf("HAVU OK")

		}

	}
}

// Here the channels are buffered, so that it is asynchronous
func SerialRead(out chan []byte) {
	msgs := []byte{}
	for {
		msg := make([]byte, 1)
		_, err := port.Read(msg)
		if err != nil {
			// log.Fatal("READ")
		}
		if msg[0] == '\n' {
			if msgs[0] == '@' {
				_msg := make([]byte, hex.DecodedLen(len(msgs)-1))
				hex.Decode(_msg, msgs[1:])
				out <- _msg
			}
			// log.Printf("Serial: %s", msgs)
			msgs = []byte{}
		} else {
			msgs = append(msgs, msg...)
		}
		os.Stdout.Write(msg)
	}
}
func MainLoop(in, out chan []byte, win, wout chan string) error {
	for {
		_pos := POS{}
		msg := <-in

		pos := _pos.FromTNH(msg)
		wout <- pos.CalculateHavu().Havu

	}
}

func main() {
	// Enable this to log into a file
	// f, err := os.OpenFile(logfile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	// if err != nil {
	// 	fmt.Printf("error opening file %s: %v", logfile, err)
	// }
	// log.SetOutput(f)
	// defer f.Close()

	// fd, err := os.OpenFile(disfile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	// if err != nil {
	// 	fmt.Printf("Error opening file %s: %v", disfile, err)
	// }
	// disfd = fd
	// defer disfd.Close() // Might report an error

	// fd, err = os.OpenFile(posfile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	// if err != nil {
	// 	fmt.Printf("Error opening file %s: %v", posfile, err)
	// }
	// posfd = fd
	// defer posfd.Close() // Might report an error

	conn := &serial.Config{Name: ser, Baud: 115200}
	_port, err := serial.OpenPort(conn)
	if err != nil {
		// log.Fatal(err)
	}
	port = _port
	defer port.Close()

	// log.Println("TACNET starting")

	sin := make(chan []byte)
	sout := make(chan []byte)
	// Buffered (hence, async) because internet might have a bad day.
	win := make(chan string, 200)
	wout := make(chan string, 18000)
	go SerialRead(sin)
	for i := 0; i < 10; i++ {
		go ToHavu(wout, win)
	}
	MainLoop(sin, sout, win, wout)

}
