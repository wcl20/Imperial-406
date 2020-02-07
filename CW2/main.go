package main

import (
	"fmt"
	"sync"
	"math/rand"
)

/**********************************
Struct for storing user information
************************************/
type User struct {
	userid string
	passwd string
}

/**********************************
Struct for storing possible actions
************************************/
type Actions struct {
	get chan chan int
	put chan int
	bye chan int
}

/****************************************************
Private channel sent from FtpThread to Authenticator
*****************************************************/
type AuthenticatorPrivate struct {
	userChannel chan User
	valid chan int
	invalid chan int
}

/****************************************************
Private channel sent from Client to FtpThread
*****************************************************/
type FtpThreadPrivate struct {
	userChannel chan User 
	sorry chan int
	welcome chan int
	actions Actions
}

/*******************************************************************************
 * CLIENT

	Client(pid) := (vs)pid<s> . s<userid, passwd> . s>{
		sorry:		0 |
		welcome:	muX. if option == 0 then s<get . s(y) . X else
						 if option == 1 then s<put . s<value> . X else
						 s<bye . 0
	}

 * Desciption:
   	The client connects to the FtpThread using a userid and password.
   	Once the client is successfully authenticated, it can continuously
   	interact with the FtpThread until it decides to terminate the session.
 		GET retrieves the object stored in FtpThread.
 		PUT alters the object stored in FtpThread.
 		BYE terminates the session.

 * Implementation:
 	User information (userid,passwd) is stored in a User struct. The userChannel
 	is used for sending user information.

 	Channel branching and channel selection is implemented using multiple
 	channels. A channel is created for each branch option, and selection is  
 	made by sending a random integer to the selected option. Therefore, 
 	the client sends to two addition channels: SORRY and WELCOME to FtpThread.

 	The Actions struct contains all actions the client can perform on FtpThread.
 	Client selects the GET option by sending a chan chan int, and use the chan int
 	to retrive the value from FtpThread. Client selection the PUT option by sending
 	an integer, the integer sent is also the value to be stored in FtpThread.

 	In reality, the Client should be able to perform any actions until it selects
 	BYE to terminate the session. However, Go Playground does not allow randomness so
 	the code predefines the options the client selects. Notice that the client selects 
 	BYE (option 2) at the 8th iteration, this is to show that once the client terminates 
 	it can no longer perform other actions.

 	You can change the client actions by modifying the options array.
 ***********************************************************************************/
func Client(pid chan<- FtpThreadPrivate, wg *sync.WaitGroup) {
	// Create private channel s and send through channel pid. 
	actions := Actions{ make(chan chan int), make(chan int), make(chan int) }
	s := FtpThreadPrivate{ make(chan User), make(chan int), make(chan int), actions }
	pid <- s
	// Send (userid,passwd) through channel s.
	s.userChannel <- User{ "wcl16", "123456" }
	// Wait for selection from FtpThread
	go func() {
		select {
		// Handle selection SORRY
		case <- s.sorry:
			fmt.Println("Invalid User")
		// Handle selection WELCOME
		case <- s.welcome:
			fmt.Println("Welcome")
			options := [10]int{0, 1, 0, 1, 0, 0, 1, 2, 1, 0}

			// Recusive process
			for k := 0; k < 10; k++ {
				option := options[k]
				if option == 0 {
					// Make selection GET
					channel := make(chan int)
					s.actions.get <- channel
					// Retrive value from channel
					y := <- channel
					fmt.Printf("Get value: %d\n", y)
				} else if option == 1 {
					// Make selection PUT
					value := rand.Intn(100)
					fmt.Printf("Put value: %d\n", value)
					s.actions.put <- value
				} else {
					// Make selection BYE
					fmt.Println("Bye")
					s.actions.bye <- 0
					break 
				}
			}
		}
		wg.Done()
	}()
} 

func Init(pid chan FtpThreadPrivate, nis chan AuthenticatorPrivate) {
	go FtpThread(pid, nis)
	go Authenticator(nis)
}

/*******************************************************************************
 * FTPTHREAD

 * Desciption:
   	The FtpThread forwards user information from clients to the authenicator. 
   	Once the user is successfully authenticated, the FtpThread starts a session
   	with the client.

 * Implementation:
 	User information (userid,passwd) is stored in a User struct. The userChannel
 	is used for sending user information.

 	Channel branching and channel selection is implemented using multiple
 	channels. A channel is created for each branch option, and selection is  
 	made by sending a random integer to the selected option. Therefore, 
 	the FtpThread sends to two addition channels: VALID and INVALID to Authenticator.

 	The FtpThread behaves differently according to the client option. If GET is
 	selected, FtpThread sends the stored variable through the received channel. If 
 	PUT is selected, FtpThread changes the stored variable x to the received integer.
 	If BYE is selected, the process terminates.

 	Infinite parallel composition is implemented using a WHILE loop. Once the FtpThread
 	receives a channel from PID, it creates a new thread immediately to process the 
 	channel. This is so that FtpThread can handle multiple clients in parallel.
 ***********************************************************************************/
func FtpThread(pid <-chan FtpThreadPrivate, nis chan<- AuthenticatorPrivate) {
	// Infinite parallel composition
	for {
		// Receive channel from channel pid
		z := <- pid
		go func(z FtpThreadPrivate) {
			// Receive (userid,passwd) from channel z
			user := <- z.userChannel
			// Create private channel s' and send through channel nis. 
			s := AuthenticatorPrivate{ make(chan User), make(chan int), make(chan int) }
			nis <- s
			// Send (userid,passwd) through channel s'
			s.userChannel <- user
			// Wait for selection from Authenticator
			select {
			// Handel selection VALID
			case <- s.valid:
				// Make selection WELCOME
				z.welcome <- 0
				x := 0
				for {
					select {
					case channel := <- z.actions.get:
						channel <- x
					case value := <- z.actions.put:
						x = value
					case <- z.actions.bye:
						return
					}
				}
			// Handel selection INVALID
			case <- s.invalid:
				// Make selection SORRY
				z.sorry <- 0
			}
		}(z)
	}
}

/*******************************************************************************
 * AUTHENTICATOR

 * Desciption:
   	The Authenticator recevies user information from FtpThread and validates the
   	user information.

 * Implementation:
 	Infinite parallel composition is implemented using a WHILE loop. Once the Authenticator
 	receives a channel from NIS, it creates a new thread immediately to process the 
 	channel. This is so that Authenticator can handle multiple FtpThreads in parallel.
 ***********************************************************************************/
func Authenticator(nis <-chan AuthenticatorPrivate) {
	for {
		// Receive channel from channel nis
		x := <- nis
		go func(x AuthenticatorPrivate) {
			// Receive (userid,passwd) from channel x
			user := <- x.userChannel
			// If password is valid ...
			pw := "123456"
			if user.passwd == pw {
				// Make selection VALID
				x.valid <- 0
			} else {
				// Make selection INVALID
				x.invalid <- 0
			}
		}(x) 
	}
}



func main() {

	var wg sync.WaitGroup
	wg.Add(1)

	pid := make(chan FtpThreadPrivate)
	nis := make(chan AuthenticatorPrivate)
	go Init(pid, nis)
	Client(pid, &wg)


	wg.Wait()
	
}

