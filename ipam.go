package goipam

type IP4AddressManager interface {
	Assign() int64
	Release(ip uint32)
}
