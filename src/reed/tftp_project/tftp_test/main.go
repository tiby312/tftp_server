package main

import "reed/tftp_project/tftp_s/tftp"
import "fmt"

func main() {
	server := tftp.CreateServer()
	go server.Run()
	_ = <-server.Finished
	fmt.Println("finished!")
}
