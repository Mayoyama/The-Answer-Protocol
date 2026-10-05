package main


// gracefulQuit sends QUIT to the server and waits briefly for a response before exiting.
func gracefulQuit(conn net.Conn, serverResponses chan string, serverErr chan error, timeout time.Duration) {
	_ = conn.SetWriteDeadline(time.Now().Add(timeout))
	_, _ = fmt.Fprintln(conn, "QUIT")

	deadline := time.After(timeout)

	for {
		select {
		case response := <-serverResponses:
			if strings.TrimSpace(response) == "OK bye" {
				_, _ = fmt.Print(response)

				return
			}

		case err := <-serverErr:
			if err != nil {
				_, _ = fmt.Printf("ERROR READING_TO_FROM_SERVER: %v.\nDISCONNECTED.\n", err)

				return
			}

		case <-deadline:
			fmt.Println("No response from server... exiting program anyway")

			return
		}
	}
}



if strings.HasPrefix(response, "OK") {

	}









conn, err := net.Dial("tcp", ":4242")

	if err != nil {
		fmt.Println("Could not connect to TCP on 4242: ", err)

		return
	}

	defer func() { _ = conn.Close() }()

	
	
	
TAPReader := bufio.NewReader(conn)

serverResponses := make(chan string)
	serverErr := make(chan error)

	quitTimeoutDuration := time.Second * 3
	commandTimeoutDuration := time.Second * 10

	go func() {
		for {
			response, err := TAPReader.ReadString('\n')
			if err != nil {
				serverErr <- err

				return
			}

			serverResponses <- response
		}
	}()

	
					_ = conn.SetWriteDeadline(time.Now().Add(commandTimeoutDuration))
				_, err = fmt.Fprintln(conn, input)

				
				
								_, _ = fmt.Fprintln(conn, "CONNECT "+input)

								
								
				if strings.TrimSpace(response) == "OK bye" {

				}
