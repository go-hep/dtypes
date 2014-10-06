package dtypes

type Type struct {
	ID      int32
	Package string
	Name    string
}

type TypeDescr struct {
	Types []Type
}
