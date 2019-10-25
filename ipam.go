package goipam

type IP4AddressManager interface {
	Assign() int64
	AssignSpecificIP(ip uint32) bool
	Release(ip uint32)
	IsIPOutOfRange(ip uint32) bool
	IsIPInRange(ip uint32) bool
}
