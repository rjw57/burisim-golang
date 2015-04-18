package main

import (
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"

	"github.com/codegangsta/cli"
	"github.com/pda/go6502/bus"
	"github.com/pda/go6502/memory"
	"github.com/pda/go6502/via6522"
	"github.com/rjw57/burisim-golang"
	"github.com/rjw57/go6502/cpu"
)

func readRamFile(bus *bus.Bus, filePath string, offset uint16) {
	log.Printf("reading %v into RAM at 0x%X", filePath, offset)
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatal("error opening file:", err)
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal("error reading file:", err)
	}

	for i, b := range data {
		bus.Write(offset+uint16(i), b)
	}
}

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
		log.Fatalf("error creating bus: %v", err)
	}

	// wire in the RAM at addresses [0x0000, 0x7fff]
	if err := cpu.Bus.Attach(&memory.Ram{}, "RAM", 0); err != nil {
		log.Fatalf("error attaching RAM: %v", err)
	}

	// load the ROM from image
	rp := args[0]
	if rom, err := memory.RomFromFile(rp); err != nil {
		log.Fatalf("error loading ROM: %v", err)
	} else {
		// wire in the ROM at addresses [0xe000, 0xffff]
		if err := cpu.Bus.Attach(rom, "ROM", 0xe000); err != nil {
			log.Fatalf("error attaching ROM: %v", err)
		}
	}

	// load the RAM from image
	rampath := c.GlobalString("load")
	if rampath != "" {
		readRamFile(cpu.Bus, rampath, 0x5000)
	}

	l, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("error listening on socket: %v", err)
	}
	acia1 := buri.CreateACIAOnListener(l)

	// attach ACIA1 at 0xdffc
	if err := cpu.Bus.Attach(acia1, "ACIA1", 0xdffc); err != nil {
		log.Fatalf("error attaching ACIA1: %v", err)
	}

	// create and attach VIA
	via1 := via6522.NewVia6522(via6522.Options{})
	if err := cpu.Bus.Attach(via1, "VIA1", 0xdfe0); err != nil {
		log.Fatalf("error attaching VIA1: %v", err)
	}

	// attach SPI master to port B
	sm := buri.SPIMaster{}
	via1.AttachToPortB(&sm)

	// attach SD card as device 0
	sd := buri.SDCardSlave{}
	sm.Slaves[0] = &sd

	// HW reset via1
	via1.Reset()

	// go forth and execute at 2MHz == 20000/(10 milliseconds)
	cpu.Reset()
	tickChan := time.Tick(10 * time.Millisecond)
	for _ = range tickChan {
		for i := 0; i < 20000; i++ {
			cpu.Step()
		}
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "buri"
	app.Usage = "emulate the Buri microcomputer"
	app.Action = runSim

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "load",
			Usage: "Specify file to load into RAM at 0x5000",
		},
	}

	app.Run(os.Args)
}
