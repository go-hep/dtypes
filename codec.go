package dtypes

type Codec struct {
	TType TypeDescr
	PType TypeDescr
	Dec   Decoder
	Enc   Encoder
}
