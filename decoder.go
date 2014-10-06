package dtypes

import (
	"io"
	"reflect"
)

type Decoder struct {
	Ops []DecInstr
}

type DecInstr func(r io.Reader, v reflect.Value) (int, error)
