package dtypes_test

import (
	"testing"

	"github.com/go-hep/dtypes"
	"github.com/go-hep/hbook"
)

func TestH1D(t *testing.T) {
	t.Skip("not ready")
	h1 := hbook.NewH1D(100, 0, 100)
	dt := dtypes.From(h1)
	if dt.ID() != 1 {
		t.Errorf("id=%v\n", dt.ID())
	}
}
