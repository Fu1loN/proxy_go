package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	en "go_proxy/encryptPass"
	"io"
	"log"
	"net"
	"slices"
	"sync"
)

func read_err(err error) {
	log.Printf("error while reading: %v", err)
}

func write_to_conn_from(readConn, writeConn net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	buffer := make([]byte, 2048)
	for {
		n, err := readConn.Read(buffer)
		if err != nil {
			if err == io.EOF {
				// log.Println("Connection closed by peer")
				return
			}
			log.Printf("error while reading data: %v", err)
			return
		}
		// log.Printf("got string %s", string(buffer[:n]))
		_, err = writeConn.Write(buffer[:n])
		if err != nil {
			log.Printf("error while writing%s", err)
		}

	}
}

// TODO store outtside
var authorize_map = map[string]string{"test_user_1": "4343502f356ecd54471a59a5dfc2084f8349e696699df723a3c3183157eed467"}

func authorize(connection net.Conn) error {
	loginSize := make([]byte, 2)

	auth_err := fmt.Errorf("Some error while authorization")
	_, err := connection.Read(loginSize)
	if err != nil {
		read_err(err)
		return auth_err
	}
	// first byte is protocol version who cares tho
	login := make([]byte, uint8(loginSize[1]))
	_, err = connection.Read(login)
	if err != nil {
		read_err(err)
		return auth_err
	}
	passSize := make([]byte, 1)
	_, err = connection.Read(passSize)
	if err != nil {
		read_err(err)
		return auth_err
	}
	password := make([]byte, uint8(passSize[0]))
	_, err = connection.Read(password)
	if err != nil {
		read_err(err)
		return auth_err
	}
	if expectedPassword, ok := authorize_map[string(login)]; ok {
		if !en.Compare(expectedPassword, password) {
			return auth_err
		}
		return nil
	}

	return auth_err
}

func handle_connection(connection net.Conn) {
	// TODO timeout on handshake
	// TODO refactor we need more functions
	defer connection.Close()

	version := make([]byte, 2)
	_, err := connection.Read(version)
	if err != nil {
		log.Printf("error while reading socks verison %v", err)
		return
	}
	// log.Println(string(version))
	if version[0] != 0x05 {
		log.Printf("wrong version %v", version[0])
		return
	}
	nmethods := version[1]
	methods := make([]byte, nmethods)
	_, err = connection.Read(methods)
	if err != nil {
		log.Printf("error while reading methods %v", err)
		return
	}

	if !slices.Contains(methods, byte(0x02)) {
		log.Printf("error methods do not contains no auth")
		return
	}
	connection.Write([]byte{0x05, 0x02})

	err = authorize(connection)
	if err != nil {
		log.Printf("Auth error %v", err)
		connection.Write([]byte{0x01, 0x01})
		return
	}
	connection.Write([]byte{0x01, 0x00})

	request := make([]byte, 4)
	_, err = connection.Read(request)
	if err != nil {
		read_err(err)
		return
	}

	// log.Println("accesible")
	command := request[1]
	addres_type := request[3]
	var address_length int
	switch addres_type {
	case 0x01:
		address_length = 4
	case 0x04:
		connection.Write([]byte{0x05, 0x08, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return
	case 0x03:
		size := make([]byte, 1)
		_, err = connection.Read(size)
		if err != nil {
			read_err(err)
			return
		}
		address_length = int(size[0])
	default:
		log.Printf("unsupported type for read")
		return
	}
	address := make([]byte, address_length)
	port := make([]byte, 2)
	_, err = connection.Read(address)
	if err != nil {
		read_err(err)
		return
	}
	_, err = connection.Read(port)
	if err != nil {
		read_err(err)
		return
	}
	// log.Printf("connecting to %v:%v", address, port)
	var connection_type string
	switch command {
	case 0x01:
		connection_type = "tcp"
	default:
		log.Printf("error unsup %v", command)
		return
	}
	port_string := fmt.Sprintf("%d", (int16(port[0])<<8 | int16(port[1])))
	fulladrrs := net.JoinHostPort(net.IP(address).String(), port_string)
	log.Println(fulladrrs)
	host_conn, err := net.Dial(connection_type, fulladrrs)
	if err != nil {
		log.Printf("error while connecting to host %s", fulladrrs)
		connection.Write([]byte{0x04})
		return
	}
	defer host_conn.Close()
	// log.Printf("succesfully connected to %v", fulladrrs)
	localAddr, ok := host_conn.LocalAddr().(*net.TCPAddr)
	if !ok {
		log.Println("something with extracting addres from connection")
		return
	}

	connection.Write([]byte{0x05, 0x00, 0x00, addres_type})
	ip4 := localAddr.IP.To4()
	portLocalAddres := make([]byte, 2)
	binary.BigEndian.PutUint16(portLocalAddres, uint16(localAddr.Port))
	_, err = connection.Write(ip4)
	if err != nil {
		log.Printf("error while writing local ip %v", err)
	}
	_, err = connection.Write(portLocalAddres)

	if err != nil {
		log.Printf("error while writing local port %v", err)
	}
	wg := sync.WaitGroup{}
	wg.Add(2)
	go write_to_conn_from(connection, host_conn, &wg)
	go write_to_conn_from(host_conn, connection, &wg)
	wg.Wait()
}

func serve() {

	server_adrr := "0.0.0.0:6969"
	lisener, err := net.Listen("tcp", server_adrr)

	if err != nil {
		log.Fatalf("error while creating listener: %s", err.Error())
	}
	log.Printf("listening to %v", server_adrr)
	defer lisener.Close()

	for {
		connection, err := lisener.Accept()
		if err != nil {
			log.Printf("error while accepting connection %s\n", err.Error())
		}
		go handle_connection(connection)

	}

}

func main() {
	encrypt := flag.Bool("e", false, "need to encrypt")
	flag.Parse()
	//TODO tf why server encrypts low priority tho it works
	if *encrypt {
		pass := flag.Arg(0)
		fmt.Printf("%x", []byte(en.EncryptString([]byte(pass))))
		return
	}
	serve()

}
