package goipam

import (
	"errors"
	"fmt"
	"strings"
)

type IP4BitmapStatus int8

const (
	IP4_BITMAP_STATUS_RUNNING IP4BitmapStatus = iota
	IP4_BITMAP_STATUS_STOPPING
	IP4_BITMAP_STATUS_STOPPED
)

type IP4Bitmap struct {
	from       uint32
	to         uint32
	count      int64
	bitmapSize int
	bitmap     []byte

	assignRequest         chan struct{}
	assignSpecificRequest chan uint32
	assignChannel         chan int64
	releaseChannel        chan uint32
	stopChannel           chan struct{}
	status                IP4BitmapStatus
}

func NewIP4BitmapFromRange(from uint32, to uint32) (*IP4Bitmap, error) {
	if from > to {
		return nil, errors.New("invalid ip range")
	}
	count := int64(to-from) + 1
	bitmapSize := int(count/8 + 1)
	ip4Bitmap := &IP4Bitmap{
		from:                  from,
		to:                    to,
		count:                 count,
		bitmapSize:            bitmapSize,
		bitmap:                make([]byte, bitmapSize),
		assignRequest:         make(chan struct{}),
		assignSpecificRequest: make(chan uint32),
		assignChannel:         make(chan int64),
		releaseChannel:        make(chan uint32),
		stopChannel:           make(chan struct{}),
		status:                IP4_BITMAP_STATUS_STOPPED,
	}

	go ip4Bitmap.handler()
	return ip4Bitmap, nil
}

func NewIP4BitmapFromStringRange(from string, to string) (*IP4Bitmap, error) {
	uint32From, err := IP2long(from)
	if err != nil {
		return nil, errors.New("NewIP4BitmapFromStringRange(): invalid start ip")
	}
	uint32To, err := IP2long(to)
	if err != nil {
		return nil, errors.New("NewIP4BitmapFromStringRange(): invalid end ip")
	}
	return NewIP4BitmapFromRange(uint32From, uint32To)
}

func NewIP4BitmapFromSubnet(subnet string) (*IP4Bitmap, error) {
	ipAndMask := strings.Split(subnet, "/")
	if len(ipAndMask) != 2 {
		return nil, errors.New("NewIP4BitmapFromSubnet(): invalid subnet format")
	}
	ip, err := IP2long(ipAndMask[0])
	if err != nil {
		return nil, err
	}

	var mask uint32
	if strings.Index(ipAndMask[1], ".") == -1 {
		var prefix1Count int
		n, err := fmt.Sscanf(ipAndMask[1], "%d", &prefix1Count)
		if n < 1 || err != nil || prefix1Count > 32 {
			return nil, errors.New("NewIP4BitmapFromSubnet(): invalid mask")
		}
		mask = 0xffffffff << (32 - prefix1Count)
	} else {
		mask, err = IP2long(ipAndMask[1])
		if err != nil {
			return nil, err
		}
	}

	return NewIP4BitmapFromRange(ip&mask, ip|(^mask))
}

func (i *IP4Bitmap) Assign() int64 {
	i.assignRequest <- struct{}{}
	return <-i.assignChannel
}

func (i *IP4Bitmap) AssignSpecificIP(ip uint32) bool {
	i.assignSpecificRequest <- ip
	return <-i.assignChannel > 0
}

func (i *IP4Bitmap) Release(ip uint32) {
	i.releaseChannel <- ip
}

func (i *IP4Bitmap) GetStatus() IP4BitmapStatus {
	return i.status
}

func (i *IP4Bitmap) Close() error {
	if i.stopChannel != nil {
		close(i.stopChannel)
		i.stopChannel = nil
	}
	return nil
}

func (i *IP4Bitmap) assign() (ip int64) {
	var bufferByte byte
	ip = -1
	for p := 0; p < i.bitmapSize; p++ {
		bufferByte = i.bitmap[p]
		if bufferByte != 0xff {
			currentBitPosition := p * 8
			for j := 0; j < 8; j++ {
				if int64(currentBitPosition)+int64(j) >= i.count {
					return
				}
				if bufferByte&1 == 0 {
					ip = int64(i.from) + int64(p)*8 + int64(j)
					i.bitmap[p] = i.bitmap[p] | byte(1<<j)
					return
				}
				bufferByte = bufferByte >> 1
			}
		}
	}
	return
}

func (i *IP4Bitmap) assignSpecificIP(ip uint32) int64 {
	if i.IsIPOutOfRange(ip) {
		return -1
	}
	_, byteIndex, bitIndex := i.calculatePosition(ip)
	if (i.bitmap[byteIndex]>>bitIndex)&1 == 0 {
		i.bitmap[byteIndex] = i.bitmap[byteIndex] | byte(1<<bitIndex)
		return int64(ip)
	}
	return -1
}

func (i *IP4Bitmap) release(ip uint32) error {
	if i.IsIPOutOfRange(ip) {
		return errors.New("ip out of range")
	}
	_, byteIndex, bitIndex := i.calculatePosition(ip)
	i.bitmap[byteIndex] = i.bitmap[byteIndex] & (^byte(1 << bitIndex))
	return nil
}

func (i *IP4Bitmap) handler() {
	i.status = IP4_BITMAP_STATUS_RUNNING
	defer func() {
		i.status = IP4_BITMAP_STATUS_STOPPED
	}()
	var ip uint32
	for {
		select {
		case <-i.assignRequest:
			i.assignChannel <- i.assign()
			break
		case ip = <-i.releaseChannel:
			_ = i.release(ip)
			break
		case ip = <-i.assignSpecificRequest:
			i.assignChannel <- i.assignSpecificIP(ip)
		case <-i.stopChannel:
			i.status = IP4_BITMAP_STATUS_STOPPING
			goto EndHandler
		}
	}
EndHandler:
}

func (i *IP4Bitmap) IsIPOutOfRange(ip uint32) bool {
	return ip < i.from || ip > i.to
}

func (i *IP4Bitmap) IsIPInRange(ip uint32) bool {
	return !i.IsIPOutOfRange(ip)
}

func (i *IP4Bitmap) GetFirst() uint32 {
	return i.from
}

func (i *IP4Bitmap) GetLast() uint32 {
	return i.from
}

func (i *IP4Bitmap) Count() uint32 {
	return uint32(i.count)
}

func (i *IP4Bitmap) calculatePosition(ip uint32) (uint32, uint32, uint32) {
	bitCount := ip - i.from
	byteIndex := bitCount / 8
	bitIndex := bitCount % 8
	return bitCount, byteIndex, bitIndex
}
