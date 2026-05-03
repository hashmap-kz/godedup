package wrapx

type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

func ToUint64[T Integer](i T) uint64 {
	if i < 0 {
		return 0
	}
	return uint64(i)
}
