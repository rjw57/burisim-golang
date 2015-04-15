package main

import (
	"fmt"
	"os"
	"unicode/utf8"

	"github.com/nsf/termbox-go"
	"github.com/pda/go6502/bus"
	"github.com/pda/go6502/cpu"
	"github.com/pda/go6502/memory"
)

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

func main() {
	// initialise termbox
	if err := termbox.Init(); err != nil {
		fmt.Errorf("error initialising termbox: %v", err)
	}
	defer termbox.Close()

	// create the Buri CPU
	cpu := cpu.Cpu{}

	// create an address bus
	if bus, err := bus.CreateBus(); err == nil {
		cpu.Bus = bus
	} else {
		fmt.Errorf("error creating bus: %v", err)
	}

	// wire in the RAM at addresses [0x0000, 0x7fff]
	if err := cpu.Bus.Attach(&memory.Ram{}, "RAM", 0); err != nil {
		fmt.Errorf("error attaching RAM: %v", err)
	}

	// load the ROM from image
	rp := os.Args[1]
	if rom, err := memory.RomFromFile(rp); err != nil {
		fmt.Errorf("error loading ROM: %v", err)
	} else {
		// wire in the ROM at addresses [0xe000, 0xffff]
		if err := cpu.Bus.Attach(rom, "ROM", 0xe000); err != nil {
			fmt.Errorf("error attaching ROM: %v", err)
		}
	}

	// channel which is closed when should exit
	exit := make(chan int)

	// create ACIA
	tx := make(chan byte)
	defer close(tx)

	acia1 := &ACIA{TxChan: tx}
	acia1.HardwareReset()

	go func() {
		buf := make([]byte, 4)
		escaped := false
		for {
			ev := termbox.PollEvent()
			switch ev.Type {
			case termbox.EventKey:
				switch {
				case escaped && ev.Ch == 'x':
					close(exit)
				case ev.Key == 1:
					// Ctrl-A
					escaped = true
				case ev.Ch != 0:
					n := utf8.EncodeRune(buf, ev.Ch)
					for i := 0; i < n; i++ {
						acia1.ReceiveByte(buf[i])
					}
				default:
					acia1.ReceiveByte(byte(ev.Key & 0xFF))
				}
			}
		}
	}()
	go func() {
		for {
			b, ok := <-tx
			if !ok {
				return
			} else {
				os.Stdout.Write([]byte{b})
			}
		}
	}()

	// attach ACIA1 at 0xdffc
	if err := cpu.Bus.Attach(acia1, "ACIA1", 0xdffc); err != nil {
		fmt.Errorf("error attaching ACIA1: %v", err)
	}

	// go forth and execute
	cpu.Reset()
	for {
		select {
		case _, ok := <-exit:
			if !ok {
				return
			}
		default:
			cpu.Step()
		}
	}
}
