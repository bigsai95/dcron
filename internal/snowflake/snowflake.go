package snowflake

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sony/sonyflake"
)

var (
	mu       sync.RWMutex
	snowNode *sonyflake.Sonyflake
)

func ConfigInit() {
	ip := getPrivateIP()
	st := sonyflake.Settings{
		StartTime: time.Date(1983, 1, 1, 0, 0, 0, 0, time.UTC),
		MachineID: lower16BitPrivateIP(ip),
	}
	snowNode = sonyflake.NewSonyflake(st)
}

func getPrivateIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		// 轉換為 *net.IPNet 並且過濾 loopback 地址
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}

	return ""
}

// Your machine ID (This case created by ip mask 2&3)
func lower16BitPrivateIP(ipString string) func() (uint16, error) {
	ip := net.ParseIP(ipString).To4()
	switch {
	case ip == nil:
		panic("parse ip err")
	case !isPrivateIPv4(ip):
		panic("ip is not private")
	}

	return func() (uint16, error) {
		return uint16(ip[2])<<8 + uint16(ip[3]), nil
	}
}

func isPrivateIPv4(ip net.IP) bool {
	return ip != nil &&
		(ip[0] == 127 || ip[0] == 10 || ip[0] == 172 && (ip[1] >= 16 && ip[1] < 32) || ip[0] == 192 && ip[1] == 168)
}

func Generate() uint64 {
	sony, err := snowNode.NextID()
	if err != nil {
		fmt.Println("Generate NextID err:", err.Error())
		return 0
	}

	return sony
}

func GenerateString() string {
	mu.Lock()
	defer mu.Unlock()
	id := Generate()

	return fmt.Sprint(id)
}
