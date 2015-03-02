package dtypes

import (
	"encoding/gob"
	"io"
	"reflect"
)

type Encoder struct {
	Ops []EncInstr
}

type EncInstr func(w io.Writer, v reflect.Value) (int, error)

type encoder struct {
	w   io.Writer
	err error
	enc *gob.Encoder
}

func newEncoder(w io.Writer) *encoder {
	return &encoder{
		w:   w,
		err: nil,
		enc: gob.NewEncoder(w),
	}
}

func (enc *encoder) encode(v interface{}) {
	if enc.err != nil {
		return
	}
	enc.err = enc.enc.Encode(v)
}
