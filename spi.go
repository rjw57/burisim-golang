package buri

import "log"

// SPIMaster emulates a parallel device which is designed to manage multiple SPI
// devices on one parallel port. The pins are arranged as follows:
//
// 	Pin	Role
// 	0	CLK
// 	1	MOSI
// 	2-4	Chip select (see below)
// 	5	Enable
// 	6	Unassigned (connected to ground)
// 	7	MISO
//
// The device supports up to 8 SPI devices (0 through 7). The device is selected
// by asserting the appropriate number on pins 2-4 (pin 2 is LSB) and taking pin
// 5 high. This causes the ~CS line for the appropriate device to be taken low.
// If pin 5 is low, no device is selected. In hardware, the device is
// implemented with a 74-series '138 3-to-8 decoder attached to pins 2 through
// 5.
//
// Send data as per the SPI spec by toggling CLK pin, setting MOSI and reading
// MISO. The precise ordering, etc depends on the SPI mode of the device.
type SPIMaster struct {
	// Slaves is an array of SPISlaves which are attached to corresponding
	// device numbers. A nil element is the absence of any slave on that
	// device. In the emulator MISO will be held low for non-existent
	// slaves but in the hardware MISO is left floating.
	Slaves [8]SPISlave

	lastOutput byte
}

type SPISlave interface {
	SetClk(bool)
	SetMOSI(bool)
	GetMISO() bool
}

func (m *SPIMaster) getSlave() SPISlave {
	// Is enable set?
	if m.lastOutput&0x20 == 0 {
		return nil
	}

	// Decode slave idx
	return m.Slaves[(m.lastOutput&0x1C)>>2]
}

func (m *SPIMaster) PinMask() byte { return 0xbf }

func (m *SPIMaster) Read() byte {
	s := m.getSlave()
	if s == nil {
		return 0
	}

	ov := m.lastOutput
	if s.GetMISO() {
		ov |= 1 << 7
	}
	return ov
}

func (m *SPIMaster) Write(b byte) {
	m.lastOutput = b
	s := m.getSlave()
	if s == nil {
		return
	}

	s.SetClk(m.lastOutput&0x01 != 0)
	s.SetMOSI(m.lastOutput&0x02 != 0)
}

func (m *SPIMaster) Shutdown()      { /* nothing */ }
func (m *SPIMaster) String() string { return "SPIMaster" }

type ByteExchanger interface {
	Exchange(byte) byte
}

// ByteExchangeSlave is a SPISlave which implements a byte-oriented exchange
// protocol.
type ByteExchangeSlave struct {
	ByteExchanger

	mosi       bool
	miso       bool
	inputByte  byte
	outputByte byte
	prevClk    bool
	bitIdx     int
}

func (s *ByteExchangeSlave) SetMOSI(v bool) { s.mosi = v }
func (s *ByteExchangeSlave) GetMISO() bool  { return s.miso }

func (s *ByteExchangeSlave) SetClk(v bool) {
	// on rising edge...
	if v && !s.prevClk {
		log.Print(s.mosi, s.miso)
		if s.bitIdx < 0 || s.bitIdx >= 8 {
			panic("invalid bit idx")
		}

		// mask of current bit, note: MSB first
		bitMask := byte(1) << uint(7-s.bitIdx)

		// record MOSI in input byte
		if s.mosi {
			s.inputByte |= bitMask
		}

		// write MISO from output byte
		s.miso = (s.outputByte & bitMask) != 0

		// increment bit index
		s.bitIdx++

		// got a full byte?
		if s.bitIdx == 8 {
			s.outputByte = s.Exchange(s.inputByte)
			s.bitIdx = 0
		}
	}

	// record clock line
	s.prevClk = v
}
