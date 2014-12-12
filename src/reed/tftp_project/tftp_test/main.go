package main

import "reed/tftp_project/tftp_s/tftp"
import "fmt"
import "os"
import "strconv"

func main() {

	aa := os.Args[1:]
	aa = aa
	var serv *tftp.Server = nil

	//if we have first arg, parse it and try and use it as port. if fail return
	if len(aa) == 1 {
		port, err := strconv.Atoi(aa[0])
		if err != nil {
			fmt.Println("failed to parse port number")
			return
		}

		serv2, ok := tftp.CreateServer(port)
		if !ok {
			fmt.Println("failed to bind port %v", port)
			return
		}
		serv = serv2
	} else { //else just use a random port
		serv = tftp.CreateServerRandPort()
	}

	go serv.Run()
	_ = <-serv.Finished
}
