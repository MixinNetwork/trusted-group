package chain

import (
	"encoding/json"
	"strconv"
)

func char_to_symbol(c byte) byte {
	if c >= 'a' && c <= 'z' {
		return (c - 'a') + 6
	}

	if c >= '1' && c <= '5' {
		return (c - '1') + 1
	}
	return 0
}

func string_to_name(str string) uint64 {
	length := len(str)
	value := uint64(0)

	for i := 0; i <= 12; i++ {
		c := uint64(0)
		if i < length && i <= 12 {
			c = uint64(char_to_symbol(str[i]))
		}
		if i < 12 {
			c &= 0x1f
			c <<= 64 - 5*(i+1)
		} else {
			c &= 0x0f
		}

		value |= c
	}

	return value
}

func S2N(s string) uint64 {
	return string_to_name(s)
}

func N2S(value uint64) string {
	charmap := []byte(".12345abcdefghijklmnopqrstuvwxyz")
	// 13 dots
	str := []byte{'.', '.', '.', '.', '.', '.', '.', '.', '.', '.', '.', '.', '.'}

	tmp := value
	for i := 0; i <= 12; i++ {
		var c byte
		if i == 0 {
			c = charmap[tmp&0x0f]
		} else {
			c = charmap[tmp&0x1f]
		}
		str[12-i] = c
		if i == 0 {
			tmp >>= 4
		} else {
			tmp >>= 5
		}
	}

	i := len(str) - 1
	for ; i >= 0; i-- {
		if str[i] != '.' {
			break
		}
	}
	return string(str[:i+1])
}

type Name struct {
	N uint64
}

func (a *Name) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

func (a *Name) UnmarshalJSON(b []byte) error {
	n, err := strconv.Unquote(string(b))
	if err != nil {
		return newError(err)
	}
	a.N = S2N(n)
	return nil
}

func NewName(s string) Name {
	return Name{N: S2N(s)}
}

func (a *Name) Pack() []byte {
	enc := NewEncoder(8)
	enc.WriteUint64(a.N)
	return enc.GetBytes()
}

func (a *Name) Unpack(data []byte) (int, error) {
	dec := NewDecoder(data)
	n, err := dec.UnpackUint64()
	if err != nil {
		return 0, err
	}
	a.N = n
	return 8, nil
}

func (t *Name) Size() int {
	return 8
}

func (a *Name) String() string {
	return N2S(a.N)
}
