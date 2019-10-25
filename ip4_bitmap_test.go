package goipam

import (
	"testing"
	"time"
)

func TestIP4Bitmap(t *testing.T) {
	from, _ := IP2long("192.168.1.0")
	to, _ := IP2long("192.168.8.250")
	ipam, err := NewIP4BitmapFromRange(from, to)
	if err != nil {
		t.Fatalf("%s", err)
	}

	testIP4Bitmap(t, from, to, ipam)


	from, _ = IP2long("192.168.0.0")
	to, _ = IP2long("192.168.1.255")
	ipam, err = NewIP4BitmapFromSubnet("192.168.1.0/23")
	if err != nil {
		t.Fatalf("%s", err)
	}

	testIP4Bitmap(t, from, to, ipam)

	ipam, err = NewIP4BitmapFromSubnet("192.168.1.0/255.255.254.0")
	if err != nil {
		t.Fatalf("%s", err)
	}

	testIP4Bitmap(t, from, to, ipam)
}

func testIP4Bitmap(t *testing.T, from uint32, to uint32, ipam *IP4Bitmap) {
	var i int64

	// assign
	for i = int64(from) ; i <= int64(to); i++ {
		ip := ipam.Assign()
		if ip < 0 {
			t.Errorf("can not assign ip %s(%d)", Long2ip(uint32(i)), i)
			return
		}
		if ip != i {
			t.Errorf("invalid assigend ip %s(%d), expected %s(%d)", Long2ip(uint32(ip)), ip, Long2ip(uint32(i)), i)
			return
		}
	}

	// out of range
	ip := ipam.Assign()
	if ip != -1 {
		t.Errorf("assigned ip out of range")
	}

	// release single ip
	ip2Release := from + ((to - from) / 2)
	ipam.Release(ip2Release)
	if ip := ipam.Assign(); ip < 0 || uint32(ip) != ip2Release {
		t.Errorf("release wrong ip")
	}

	// release all
	for i := from ; i <= to; i++ {
		ipam.Release(i)
	}

	// assign all again
	for i = int64(from) ; i <= int64(to); i++ {
		ip := ipam.Assign()
		if ip < 0 {
			t.Errorf("can not assign ip %s(%d)", Long2ip(uint32(i)), i)
			return
		}
		if ip != i {
			t.Errorf("invalid assigend ip %s(%d), expected %s(%d)", Long2ip(uint32(ip)), ip, Long2ip(uint32(i)), i)
			return
		}
	}

	// out of range again
	ip = ipam.Assign()
	if ip != -1 {
		t.Errorf("assigned ip out of range")
	}

	_ = ipam.Close()
	time.Sleep(1 * time.Second)
	if ipam.GetStatus() != IP4_BITMAP_STATUS_STOPPED {
		t.Errorf("ipam can not be stopeed")
	}
}