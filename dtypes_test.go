package dtypes_test

import (
	"reflect"
	"testing"

	"github.com/go-hep/dtypes"
	"github.com/go-hep/hbook"
)

const (
	idH1D dtypes.ID = 2363885965
)

func TestH1D(t *testing.T) {
	h1 := hbook.NewH1D(100, 0, 100)
	want := struct {
		name  string
		id    dtypes.ID
		v     interface{}
		descr []dtypes.Descr
	}{
		name:  "github.com/go-hep/hbook.H1D",
		id:    idH1D,
		v:     h1,
		descr: nil,
	}
	dt := dtypes.New(h1)

	if dt.ID() != want.id {
		t.Errorf("h1d: invalid ID. got=%d. want=%d\n", dt.ID(), want.id)
	}

	if dt.Name() != want.name {
		t.Errorf("h1d: invalid name. got=%q. want=%q\n", dt.Name(), want.name)
	}

	if !reflect.DeepEqual(dt, want.id.Type()) {
		t.Errorf("name=%q - invalid Type. got=%v want=%v\n",
			want.name, dt, want.id.Type(),
		)
	}

	if !reflect.DeepEqual(dt.Descr(), want.descr) {
		t.Errorf("name=%q - invalid Descr. got=%+v. want=%+v\ndt: %#v\n",
			want.name, dt.Descr(), want.descr, dt,
		)
	}
}
