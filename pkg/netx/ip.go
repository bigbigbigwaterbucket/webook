package netx

import "net"

// 需要联网才能拿到外网出口ip!
func GetOutIp() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	defer conn.Close()
	if err != nil {
		return ""
	}
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
