package main

import (
	"fmt"
	"os"
	"unicode/utf8"

	"github.com/nsf/termbox-go"
	"github.com/pda/go6502/bus"
	"github.com/pda/go6502/cpu"
	"github.com/pda/go6502/memory"
	"github.com/rjw57/burisim-golang"
)

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

	acia1 := &buri.ACIA{TxChan: tx}
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
