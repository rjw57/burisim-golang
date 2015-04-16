package buri

import "fmt"

type ACIA struct {
	// CommandReg is a copy of the current command register
	CommandReg byte

	// ControlReg is a copy of the current control register
	ControlReg byte

	// StatusReg is a copy of the current status register
	StatusReg byte

	// RecvReg is the received data register
	RecvReg byte

	// TxChan is a channel which has a byte written to it when the ACIA is
	// ready to transmit
	TxChan chan byte
}

func (*ACIA) Shutdown() {}
func (*ACIA) Size() int { return 4 }

func (a *ACIA) HardwareReset() {
	a.ControlReg = 0x00
	a.CommandReg = 0x02
	a.StatusReg = 0x10
}

func (a *ACIA) ReceiveByte(v byte) {
	a.RecvReg = v
	if a.StatusReg&0x08 != 0 {
		// there was overrun
		a.StatusReg |= 0x04
	}
	a.StatusReg |= 0x08 // recv. reg full
}

func (a *ACIA) Read(i uint16) byte {
	switch i {
	case 0: // recv. reg
		a.StatusReg &= 0xF7 // clear recv. reg full flag
		return a.RecvReg
	case 1: // status reg
		return a.StatusReg
	case 2: // cmd. reg
		return a.CommandReg
	case 3: // control reg
		return a.ControlReg
	}
	return 0
}

func (a *ACIA) Write(i uint16, v byte) {
	switch i {
	case 0:
		a.TxChan <- v
	case 1: // programmed reset
		a.CommandReg &= 0x1F
		a.CommandReg |= 0x02
		a.StatusReg &= 0xFB
	case 2: // cmd. reg
		a.CommandReg = v
	case 3: // control reg
		a.ControlReg = v
	default:
		fmt.Printf("write @ %v, %v\n", i, v)
	}
}
