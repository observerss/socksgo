package main

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"os"

	"github.com/urfave/cli"
)

const (
	BufSize = 4096
)

type SocksProxy struct {
	ListenAddr  *net.TCPAddr
	BindAddr    *net.TCPAddr
	Username    string
	Password    string
	AllowNoAuth bool
}

func (sp *SocksProxy) RunServer() error {
	log.Println("Listening On", sp.ListenAddr.String(), "Outgoing From", sp.BindAddr.IP.String())
	sock, err := net.ListenTCP("tcp", sp.ListenAddr)
	if err != nil {
		return err
	}

	defer sock.Close()

	for {
		conn, err := sock.AcceptTCP()
		if err != nil {
			log.Println(err)
			continue
		}
		// drop data when closed
		conn.SetLinger(0)
		go sp.handleConn(conn)
	}
	return nil
}

func (sp *SocksProxy) handleConn(conn *net.TCPConn) {
	log.Println("Accepted connection from", conn.RemoteAddr())
	defer conn.Close()

	buf := make([]byte, 512)

	// start parsing rfc1928 socks5 protocol
	/** Recv1
	+----+----------+----------+
	|VER | NMETHODS | METHODS  |
	+----+----------+----------+
	| 1  |    1     | 1 to 255 |
	+----+----------+----------+
	*/
	_, err := conn.Read(buf)
	if err != nil {
		return
	}

	// only version 5 is supported
	if buf[0] != 0x05 {
		return
	}

	// should support socks5 auth
	support_noauth := false
	support_auth := false
	for _, v := range buf[2 : 2+buf[1]] {
		if v == 0x02 {
			support_auth = true
		}
		if v == 0x00 {
			support_noauth = true
		}
	}

	/** Send1
	+----+--------+
	|VER | METHOD |
	+----+--------+
	| 1  |   1    |
	+----+--------+
	*/
	switch {
	case sp.AllowNoAuth && support_noauth:
		// no auth is required
		conn.Write([]byte{0x05, 0x00})

	case support_auth:
		// auth required, expecting rfc1929
		conn.Write([]byte{0x05, 0x02})
		/** Recv2
		+----+------+----------+------+----------+
		|VER | ULEN |  UNAME   | PLEN |  PASSWD  |
		+----+------+----------+------+----------+
		| 1  |  1   | 1 to 255 |  1   | 1 to 255 |
		+----+------+----------+------+----------+
		*/
		_, err = conn.Read(buf)
		if err != nil {
			return
		}

		// auth version should be 1
		if buf[0] != 0x01 {
			conn.Write([]byte{0x01, 0x01})
			return
		}

		// auth should pass
		len1 := buf[1]
		len2 := buf[2+len1]
		username := string(buf[2 : 2+len1])
		password := string(buf[3+len1 : 3+len1+len2])
		if username != sp.Username || password != sp.Password {
			conn.Write([]byte{0x01, 0x01})
			return
		}

		/** Send2
		+----+--------+
		|VER | STATUS |
		+----+--------+
		| 1  |   1    |
		+----+--------+
		*/
		conn.Write([]byte{0x01, 0x00})

	default:
		// no method available, quit
		conn.Write([]byte{0x01, 0xFF})
		return
	}

	/** Recv3
	+----+-----+-------+------+----------+----------+
	|VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	+----+-----+-------+------+----------+----------+
	| 1  |  1  | X'00' |  1   | Variable |    2     |
	+----+-----+-------+------+----------+----------+
	*/
	_, err = conn.Read(buf)
	if err != nil {
		return
	}

	// yes i check socks version again
	if buf[0] != 0x05 {
		conn.Write([]byte{0x05, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return
	}

	// only 'CONNECT' CMD is allowed here
	if buf[1] != 0x01 {
		conn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return
	}

	var ip []byte
	var port []byte
	switch buf[3] {
	case 0x01:
		// IP V4 address
		ip = buf[4 : 4+net.IPv4len]
		port = buf[4+net.IPv4len : 6+net.IPv4len]
	case 0x03:
		// DOMAINNAME
		len3 := buf[4]
		ipAddr, err := net.ResolveIPAddr("ip", string(buf[5:5+len3]))
		if err != nil {
			conn.Write([]byte{0x05, 0x03, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
			return
		}
		ip = ipAddr.IP
		port = buf[5+len3 : 7+len3]
	case 0x04:
		// IP V6 address
		ip = buf[4 : 4+net.IPv6len]
		port = buf[4+net.IPv6len : 6+net.IPv6len]
	default:
		// address type not supported
		conn.Write([]byte{0x05, 0x08, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return
	}

	remoteAddr := &net.TCPAddr{
		IP:   ip,
		Port: int(binary.BigEndian.Uint16(port)),
	}

	// Connect to Remote
	remoteConn, err := net.DialTCP("tcp", sp.BindAddr, remoteAddr)
	if err != nil {
		conn.Write([]byte{0x05, 0x04, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return
	} else {
		defer remoteConn.Close()
		remoteConn.SetLinger(0)
		localAddr := remoteConn.LocalAddr().(*net.TCPAddr)
		portb := make([]byte, 2)
		binary.BigEndian.PutUint16(portb, uint16(localAddr.Port))
		resp := append([]byte{0x05, 0x00, 0x00, 0x01}, localAddr.IP...)
		resp = append(resp, portb...)

		/** Send3
		+----+-----+-------+------+----------+----------+
		|VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
		+----+-----+-------+------+----------+----------+
		| 1  |  1  | X'00' |  1   | Variable |    2     |
		+----+-----+-------+------+----------+----------+
		*/
		conn.Write(resp)
	}

	// Now do the forwarding
	go io.Copy(conn, remoteConn)
	io.Copy(remoteConn, conn)
}

func main() {
	app := cli.NewApp()
	app.Name = "Socksgo"
	app.Usage = "a minimal socks5 server that can switch outgoing ip"
	app.Version = "0.1"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "host",
			Value: "0.0.0.0",
			Usage: "host ip to listen on",
		},
		cli.StringFlag{
			Name:  "port",
			Value: "11080",
			Usage: "host port to listen on",
		},
		cli.StringFlag{
			Name:  "username",
			Value: "",
			Usage: "socks5 username, optional",
		},
		cli.StringFlag{
			Name:  "password",
			Value: "",
			Usage: "socks5 password, optional",
		},
		cli.StringFlag{
			Name:  "eip",
			Value: "0.0.0.0",
			Usage: "external ip to bind on",
		},
	}
	app.Action = func(c *cli.Context) error {
		listenAddr, err := net.ResolveTCPAddr("tcp", c.String("host")+":"+c.String("port"))
		if err != nil {
			log.Fatal(err)
		}
		bindAddr, err := net.ResolveTCPAddr("tcp", c.String("eip")+":0")
		if err != nil {
			log.Fatal(err)
		}
		app := &SocksProxy{
			ListenAddr:  listenAddr,
			BindAddr:    bindAddr,
			Username:    c.String("username"),
			Password:    c.String("password"),
			AllowNoAuth: c.String("username") == "",
		}
		return app.RunServer()
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
