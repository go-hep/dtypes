package dtypes

import (
	"encoding/gob"
	"io"
	"reflect"
)

type Decoder struct {
	Ops []DecInstr
}

type DecInstr func(r io.Reader, v reflect.Value) (int, error)

type decoder struct {
	r   io.Reader
	err error
	dec *gob.Decoder
}

func newDecoder(r io.Reader) *decoder {
	return &decoder{
		r:   r,
		err: nil,
		dec: gob.NewDecoder(r),
	}
}

func (dec *decoder) decode(v interface{}) {
	if dec.err != nil {
		return
	}
	dec.err = dec.dec.Decode(v)
}
