package buri

import "log"

// SDCardSlave emulates an SD card in SPI mode. The SPI interface for SD cards
// is "mode 0" (i.e. CPOL = CPHA = 0). In other words, data is captured on CLK
// rising edge and propagated (or processed) on the falling edge. The protocol
// is byte oriented with data being sent MSB first. Multi-byte values are also
// MSB first.
type SDCardSlave struct {
	ByteExchangeSlave

	Data []byte
}

const (
	CmdGoIdleState        = 0
	CmdSendOpCond         = 1
	CmdSendIfCond         = 8
	CmdSendCSD            = 9
	CmdSendCID            = 10
	CmdStopTransmission   = 12
	CmdSetBlockLen        = 16
	CmdReadSingleBlock    = 17
	CmdReadMultipleblock  = 18
	CmdSetBlockCount      = 23
	CmdWriteBlock         = 24
	CmdWriteMultipleBlock = 25
	CmdAppCmd             = 55
	CmdReadOCR            = 58

	ACmdAppSendOpCond        = 41
	ACmdSetWrBlockEraseCount = 23
)

func (sd *SDCardSlave) Exchange(input byte) byte {
	log.Print("input:", input)
	return 0
}
