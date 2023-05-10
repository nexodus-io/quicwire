package quicnet

import (
	"encoding/json"

	"github.com/quic-go/quic-go"
	"github.com/songgao/water"
)

type Ctx struct {
	localIf *water.Interface
	quic.Connection
	Data []byte
}

func (ctx *Ctx) String() string {
	return string(ctx.Data)
}

func (ctx *Ctx) Parse(data any) error {
	err := json.Unmarshal(ctx.Data, data)
	return err
}

func (ctx *Ctx) Send(data string) error {
	err := ctx.SendMessage([]byte(data))
	return err
}

func (ctx *Ctx) SendBytes(data []byte) error {
	err := ctx.SendMessage(data)
	return err
}

func (ctx *Ctx) SendJson(data any) error {
	res, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return ctx.SendMessage(res)
}
