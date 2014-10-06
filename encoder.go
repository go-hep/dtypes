package dtypes

import (
	"io"
	"reflect"
)

type Encoder struct {
	Ops []EncInstr
}

type EncInstr func(w io.Writer, v reflect.Value) (int, error)
