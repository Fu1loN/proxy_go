package main

import (
	"encoding/binary"
	"fmt"
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
				log.Println("Connection closed by peer")
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

func handle_connection(connection net.Conn) {
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

	// TODO uprgade to auth
	if !slices.Contains(methods, byte(0x00)) {
		log.Printf("error methods do not contains no auth")
		return
	}
	connection.Write([]byte{0x05, 0x00})
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
		address_length = 16
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
	log.Printf("connecting to %v:%v", address, port)
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
		log.Println("error while connecting to host %v", fulladrrs)
		connection.Write([]byte{0x04})
		return
	}
	defer host_conn.Close()
	log.Printf("succesfully connected to %v", fulladrrs)
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

func main() {
	log.Println("Hello World!!!")
	server_adrr := "130.193.50.203:6969"
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
