package dtypes

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"reflect"
	"testing"
)

type dataTypeI struct {
	I int
}

type dataTypeF struct {
	F float64
}

var dtypes = []struct {
	name string
	id   ID
	v    interface{}
}{
	{
		name: "bool",
		id:   idBool,
		v:    bool(false),
	},
	{
		name: "int",
		id:   idInt,
		v:    int(0),
	},
	{
		name: "int8",
		id:   idInt8,
		v:    int8(0),
	},
	{
		name: "int16",
		id:   idInt16,
		v:    int16(0),
	},
	{
		name: "int32",
		id:   idInt32,
		v:    int32(0),
	},
	{
		name: "int64",
		id:   idInt64,
		v:    int64(0),
	},
	{
		name: "uint",
		id:   idUint,
		v:    uint(0),
	},
	{
		name: "uint8",
		id:   idUint8,
		v:    uint8(0),
	},
	{
		name: "uint16",
		id:   idUint16,
		v:    uint16(0),
	},
	{
		name: "uint32",
		id:   idUint32,
		v:    uint32(0),
	},
	{
		name: "uint64",
		id:   idUint64,
		v:    uint64(0),
	},
	{
		name: "float32",
		id:   idFloat32,
		v:    float32(0),
	},
	{
		name: "float64",
		id:   idFloat64,
		v:    float64(0),
	},
	{
		name: "bytes",
		id:   idBytes,
		v:    []byte{0},
	},
	{
		name: "string",
		id:   idString,
		v:    "",
	},
	{
		name: "complex64",
		id:   idComplex64,
		v:    complex64(0),
	},
	{
		name: "complex128",
		id:   idComplex128,
		v:    complex128(0),
	},
	{
		name: "interface",
		id:   idInterface,
		v: func() interface{} {
			rt := reflect.TypeOf((*interface{})(nil)).Elem()
			return reflect.New(rt).Elem().Interface()
		}(),
	},
	{
		name: "github.com/go-hep/dtypes.dataTypeI",
		id:   3772976347,
		v:    dataTypeI{},
	},
	{
		name: "github.com/go-hep/dtypes.dataTypeF",
		id:   3772779736,
		v:    dataTypeF{},
	},
}

func TestTypeOf(t *testing.T) {
	for _, table := range dtypes {
		dt := New(table.v)
		if dt.ID() != table.id {
			t.Errorf("name=%q - invalid ID. got=%d want=%d\n",
				table.name, dt.ID(), table.id,
			)
		}

		if dt.Name() != table.name {
			t.Errorf("name=%q - invalid name. got=%q want=%q\n",
				table.name, dt.Name(), table.name,
			)
		}

		if !reflect.DeepEqual(dt, table.id.Type()) {
			t.Errorf("name=%q - invalid Type. got=%v want=%v\n",
				table.name, dt, table.id.Type(),
			)
		}
	}
}

func TestTypeRW(t *testing.T) {
	for _, table := range dtypes {
		fmt.Printf("--- [%s] ---\n", table.name)
		dt := New(table.v)
		buf := new(bytes.Buffer)
		err := gob.NewEncoder(buf).Encode(dt)
		if err != nil {
			t.Errorf("name=%q: error encoding dtype (err=%v)\n", table.name, err)
		}

		//var rt interface{} = reflect.New(reflect.TypeOf(dt)).Elem().Addr()
		rt := reflect.New(reflect.ValueOf(dt).Type())
		err = gob.NewDecoder(buf).Decode(rt.Interface())
		if err != nil {
			t.Errorf("name=%q: error decoding dtype (err=%v)\n", table.name, err)
		}

		if !reflect.DeepEqual(dt, rt.Elem().Interface()) {
			t.Errorf("name=%q - r/w error. want=%#v got=%#v\n", table.name, dt, rt.Elem().Interface())
		}
	}
}
