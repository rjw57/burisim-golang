package main

import (
	"fmt"
	"net"
	"os"

	"github.com/codegangsta/cli"
	"github.com/pda/go6502/bus"
	"github.com/pda/go6502/cpu"
	"github.com/pda/go6502/memory"
	"github.com/rjw57/burisim-golang"
)

func runSim(c *cli.Context) {
	args := c.Args()
	if len(args) == 0 {
		panic("no ROM image specified")
	}

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
	rp := args[0]
	if rom, err := memory.RomFromFile(rp); err != nil {
		fmt.Errorf("error loading ROM: %v", err)
	} else {
		// wire in the ROM at addresses [0xe000, 0xffff]
		if err := cpu.Bus.Attach(rom, "ROM", 0xe000); err != nil {
			fmt.Errorf("error attaching ROM: %v", err)
		}
	}

	l, err := net.Listen("tcp", ":9000")
	if err != nil {
		fmt.Errorf("error listening on socket: %v", err)
	}
	acia1 := buri.CreateACIAOnListener(l)

	// attach ACIA1 at 0xdffc
	if err := cpu.Bus.Attach(acia1, "ACIA1", 0xdffc); err != nil {
		fmt.Errorf("error attaching ACIA1: %v", err)
	}

	// go forth and execute
	cpu.Reset()
	for {
		cpu.Step()
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "buri"
	app.Usage = "emulate the Buri microcomputer"
	app.Action = runSim
	app.Run(os.Args)
}
