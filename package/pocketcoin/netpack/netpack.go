package netpack


import (
	"fmt"
	"pocketcoin/coin"
	"net"
	"bufio"
	"encoding/json"
)


func check(err error) {
	if err != nil {
		panic(err)
	}
}


func ConstructNetworkPacket(header coin.RequestHeader, body string) coin.NetworkPacket {
	netPacket := coin.NetworkPacket{}
	netPacket.Header = header
	netPacket.Body = body

	return netPacket
}


func ConstructRequestHeader(node string, request string) coin.RequestHeader {
	reqHeader := coin.RequestHeader{}
	reqHeader.Node = node
	reqHeader.Request = request

	return reqHeader
}


func DeserialisePacket(packetString string) coin.NetworkPacket {
	packet := coin.NetworkPacket{}
	json.Unmarshal([]byte(packetString), &packet)

	return packet
}


func BroadcastPacket(packetString string, port string) {
	fmt.Println("Attempting connection to node", port)
	conn, err := net.Dial("tcp", "localhost:"+port)
	if err == nil {
		fmt.Fprintf(conn, packetString + "\n")
		fmt.Println("Packet send successfully to node", port)
		conn.Close()
	}
}


func BroadcastDuplexPacket(packetString string, port string) (bool, coin.NetworkPacket) {
	fmt.Println("Attempting duplex connection to node...")
	conn, err := net.Dial("tcp", "localhost:"+port)
	if err == nil {
		fmt.Fprintf(conn, packetString + "\n")
		fmt.Println("Packet send successfully to node!")

		recv, _ := bufio.NewReader(conn).ReadString('\n')
		conn.Close()
		return true, DeserialisePacket(recv)

	} else {
		return false, coin.NetworkPacket{}
	}
}