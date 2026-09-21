package stunserver

import (
	"encoding/binary"
	"net"
	"sync"
)

const (
	bindingRequest = 0x0001
	bindingSuccess = 0x0101
	magicCookie    = 0x2112a442
	attrXORMapped  = 0x0020
	attrOther      = 0x802c
	stunHeaderSize = 20
)

// Server is a loopback STUN server used by e2e tests so NAT discovery does not
// depend on a public STUN endpoint.
type Server struct {
	primary   *net.UDPConn
	alternate *net.UDPConn
	done      chan struct{}
	wg        sync.WaitGroup
}

func New() (*Server, error) {
	primary, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		return nil, err
	}
	alternate, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		primary.Close()
		return nil, err
	}
	return &Server{
		primary:   primary,
		alternate: alternate,
		done:      make(chan struct{}),
	}, nil
}

func (s *Server) Addr() string {
	return s.primary.LocalAddr().String()
}

func (s *Server) Run() {
	s.serve(s.primary, true)
	s.serve(s.alternate, false)
}

func (s *Server) Close() {
	close(s.done)
	s.primary.Close()
	s.alternate.Close()
	s.wg.Wait()
}

func (s *Server) serve(conn *net.UDPConn, includeOther bool) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		buf := make([]byte, 1024)
		for {
			n, src, err := conn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			select {
			case <-s.done:
				return
			default:
			}
			response, err := bindingSuccessResponse(buf[:n], src, includeOther, s.alternate.LocalAddr().(*net.UDPAddr))
			if err != nil {
				continue
			}
			_, _ = conn.WriteToUDP(response, src)
		}
	}()
}

func bindingSuccessResponse(request []byte, src *net.UDPAddr, includeOther bool, other *net.UDPAddr) ([]byte, error) {
	if len(request) < stunHeaderSize || binary.BigEndian.Uint16(request[0:2]) != bindingRequest {
		return nil, net.InvalidAddrError("invalid stun request")
	}

	attrs := [][]byte{xorMappedIPv4(src.IP, src.Port)}
	if includeOther {
		attrs = append(attrs, otherIPv4(other.IP, other.Port))
	}

	length := 0
	for _, attr := range attrs {
		length += len(attr)
	}
	response := make([]byte, stunHeaderSize, stunHeaderSize+length)
	binary.BigEndian.PutUint16(response[0:2], bindingSuccess)
	binary.BigEndian.PutUint16(response[2:4], uint16(length))
	binary.BigEndian.PutUint32(response[4:8], magicCookie)
	copy(response[8:20], request[8:20])
	for _, attr := range attrs {
		response = append(response, attr...)
	}
	return response, nil
}

func xorMappedIPv4(ip net.IP, port int) []byte {
	ip4 := ip.To4()
	if ip4 == nil {
		ip4 = net.IPv4(127, 0, 0, 1).To4()
	}
	value := make([]byte, 8)
	value[1] = 0x01
	binary.BigEndian.PutUint16(value[2:4], uint16(port)^uint16(magicCookie>>16))
	copy(value[4:], ip4)
	for i := range 4 {
		value[4+i] ^= byte(uint32(magicCookie) >> uint(24-8*i))
	}
	return stunAttr(attrXORMapped, value)
}

func otherIPv4(ip net.IP, port int) []byte {
	ip4 := ip.To4()
	value := make([]byte, 8)
	value[1] = 0x01
	binary.BigEndian.PutUint16(value[2:4], uint16(port))
	copy(value[4:], ip4)
	return stunAttr(attrOther, value)
}

func stunAttr(typ uint16, value []byte) []byte {
	padded := (len(value) + 3) &^ 3
	attr := make([]byte, 4+padded)
	binary.BigEndian.PutUint16(attr[0:2], typ)
	binary.BigEndian.PutUint16(attr[2:4], uint16(len(value)))
	copy(attr[4:], value)
	return attr
}
