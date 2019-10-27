package goipam

import "io"

type IP4AddressManager interface {
	Assign() int64
	AssignSpecificIP(ip uint32) bool
	Release(ip uint32)
	IsIPOutOfRange(ip uint32) bool
	IsIPInRange(ip uint32) bool
	GetFirst() uint32
	GetLast() uint32
	Count() uint32
	io.Closer
}
