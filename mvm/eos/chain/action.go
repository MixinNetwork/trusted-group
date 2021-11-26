package chain

import "unsafe"

type PermissionLevel struct {
	Actor      Name `json:"actor"`
	Permission Name `json:"permission"`
}

func (t *PermissionLevel) Pack() []byte {
	enc := NewEncoder(16)
	enc.Pack(&t.Actor)
	enc.Pack(&t.Permission)
	return enc.GetBytes()
}

func (t *PermissionLevel) Unpack(data []byte) (int, error) {
	dec := NewDecoder(data)
	dec.Unpack(&t.Actor)
	dec.Unpack(&t.Permission)
	return dec.Pos(), nil
}

func (t *PermissionLevel) Size() int {
	return 16
}

type Action struct {
	Account       Name              `json:"account"`
	Name          Name              `json:"name"`
	Authorization []PermissionLevel `json:"authorization"`
	Data          Bytes             `json:"data"`
}

func NewAction(perm PermissionLevel, account Name, name Name, args ...interface{}) *Action {
	a := &Action{}
	a.Account = account
	a.Name = name

	a.Authorization = append(a.Authorization, perm)
	if len(args) == 0 {
		a.Data = []byte{}
		return a
	}

	size := 0
	for _, v := range args {
		n, err := CalcPackedSize(v)
		if err != nil {
			panic(err.Error())
		}
		size += n
	}
	enc := NewEncoder(size)
	for _, arg := range args {
		enc.Pack(arg)
	}
	a.Data = enc.GetBytes()
	return a
}

func PackUint64(n uint64) []byte {
	p := [8]byte{}
	pp := (*[8]byte)(unsafe.Pointer(&n))
	copy(p[:], pp[:])
	return p[:]
}

func PackArray(a []Serializer) []byte {
	buf := []byte{byte(len(a))}
	for _, v := range a {
		buf = append(buf, v.Pack()...)
	}
	return buf
}

func (a *Action) SetData(data []byte) {
	a.Data = data
}

func (a *Action) Pack() []byte {
	enc := NewEncoder(a.Size())
	enc.PackName(a.Account)
	enc.PackName(a.Name)
	enc.PackLength(len(a.Authorization))
	for _, v := range a.Authorization {
		enc.Pack(&v)
	}
	enc.Pack(([]byte)(a.Data))
	return enc.GetBytes()
	// buf := []byte{}
	// buf = append(buf, PackUint64(a.Account)...)
	// buf = append(buf, PackUint64(a.Name)...)

	// buf = append(buf, PackUint32(uint32(len(a.Authorization)))...)
	// for _, v := range a.Authorization {
	// 	buf = append(buf, v.Pack()...)
	// }

	// buf = append(buf, a.Data.Pack()...)
	// return buf
}

func (a *Action) Unpack(b []byte) (int, error) {
	dec := NewDecoder(b)
	dec.Unpack(&a.Account)
	dec.Unpack(&a.Name)
	length, err := dec.UnpackLength()
	if err != nil {
		return 0, err
	}
	a.Authorization = make([]PermissionLevel, length)
	for i := 0; i < length; i++ {
		dec.Unpack(&a.Authorization[i])
	}
	dec.Unpack(&a.Data)
	return dec.Pos(), nil
}

func (a *Action) Size() int {
	return 8 + 8 + 5 + len(a.Authorization)*8 + 5 + len(a.Data)
}

func (a *Action) AddPermission(actor Name, permission Name) {
	a.Authorization = append(a.Authorization, PermissionLevel{actor, permission})
}
