package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/rjw57/go6502"
)

const (
	MemoryReadMessageType         = "read"
	MemoryReadResponseMessageType = "readresp"
	MemoryWriteMessageType        = "write"
)

type BusWebSocketHandler struct {
	Bus              *go6502.Bus
	Upgrader         websocket.Upgrader
	ReadRequestChan  chan ReadRequest
	WriteRequestChan chan WriteRequest
}

type MemoryReadResponseMessage struct {
	Address uint16
	Value   byte
}

type MemoryWriteMessage struct {
	Address uint16
	Value   byte
}

type MemoryReadMessage struct {
	Address uint16
}

type Message struct {
	MemoryWriteMessage        *MemoryWriteMessage
	MemoryReadMessage         *MemoryReadMessage
	MemoryReadResponseMessage *MemoryReadResponseMessage
}

type typedMessage struct {
	Type string `json:"_type"`
}

func (m MemoryReadMessage) MarshalJSON() ([]byte, error) {
	f := make(map[string]interface{})
	f["_type"] = MemoryReadMessageType
	f["address"] = m.Address
	return json.Marshal(f)
}

func (m MemoryWriteMessage) MarshalJSON() ([]byte, error) {
	f := make(map[string]interface{})
	f["_type"] = MemoryWriteMessageType
	f["address"] = m.Address
	f["value"] = m.Value
	return json.Marshal(f)
}

func (m *Message) UnmarshalJSON(b []byte) error {
	var err error

	// unmarshal message type
	t := typedMessage{}
	if err = json.Unmarshal(b, &t); err != nil {
		return err
	}
	switch t.Type {
	case MemoryWriteMessageType:
		p := MemoryWriteMessage{}
		if err = json.Unmarshal(b, &p); err != nil {
			return err
		}
		m.MemoryWriteMessage = &p
	case MemoryReadMessageType:
		p := MemoryWriteMessage{}
		if err = json.Unmarshal(b, &p); err != nil {
			return err
		}
		m.MemoryWriteMessage = &p
	case MemoryReadResponseMessageType:
		p := MemoryReadResponseMessage{}
		if err = json.Unmarshal(b, &p); err != nil {
			return err
		}
		m.MemoryReadResponseMessage = &p
	default:
		return errors.New("unknown message type: " + t.Type)
	}

	return nil
}

func (h BusWebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// channel for writing to client
	wc := make(chan interface{})
	done := make(chan int, 0)
	defer func() {
		done <- 1
		close(done)
		close(wc)
	}()

	// write pump
	go func(c chan interface{}) {
		for msg := range c {
			if err := conn.WriteJSON(&msg); err != nil {
				log.Print("cannot write: ", err)
				return
			}
		}
	}(wc)

	var curreq *ReadRequest

	// handler for write requests
	go func() {
		for {
			select {
			case <-done:
				return
			case wr := <-h.WriteRequestChan:
				msg := MemoryWriteMessage{
					Address: wr.Address,
					Value:   wr.Value,
				}
				wc <- msg
			case rr := <-h.ReadRequestChan:
				curreq = &rr
				msg := MemoryReadMessage{
					Address: rr.Address,
				}
				wc <- msg
			}
		}
	}()

	// read pump
	for {
		msg := Message{}
		err = conn.ReadJSON(&msg)
		if err != nil {
			log.Print("cannot read message: ", err)
			return
		}

		switch {
		case msg.MemoryReadResponseMessage != nil:
			m := msg.MemoryReadResponseMessage
			if m.Address != curreq.Address {
				log.Print(
					"skipping read response for ",
					m.Address)
			} else {
				curreq.Reply <- m.Value
			}
		}
	}
}

type ReadRequest struct {
	Address uint16
	Reply   chan byte
}

type WriteRequest struct {
	Address uint16
	Value   byte
}

func NewBusWebSocketHandler() *BusWebSocketHandler {
	return &BusWebSocketHandler{
		ReadRequestChan:  make(chan ReadRequest),
		WriteRequestChan: make(chan WriteRequest),
	}
}

func (h BusWebSocketHandler) Read(addr uint16) byte {
	r := make(chan byte)
	h.ReadRequestChan <- ReadRequest{
		Address: addr, Reply: r,
	}
	return <-r
}

func (h BusWebSocketHandler) Write(addr uint16, value byte) {
	h.WriteRequestChan <- WriteRequest{
		Address: addr, Value: value,
	}
}

type BusWebSocketMemory struct {
	Handler *BusWebSocketHandler
	Offset  uint16
	Length  uint16
}

func (h *BusWebSocketHandler) NewBusWebSocketMemory(offset uint16, size uint16) *BusWebSocketMemory {
	rv := &BusWebSocketMemory{
		Handler: h,
		Offset:  offset,
		Length:  size,
	}
	return rv
}

func (m BusWebSocketMemory) Shutdown() {}

func (m BusWebSocketMemory) Size() int {
	return int(m.Length)
}

func (m BusWebSocketMemory) Read(a uint16) byte {
	if a >= m.Length {
		panic("invalid address in read")
	}

	return m.Handler.Read(a + m.Offset)
}

func (m BusWebSocketMemory) Write(a uint16, v byte) {
	if a >= m.Length {
		panic("invalid address in read")
	}

	m.Handler.Write(a+m.Offset, v)
}
