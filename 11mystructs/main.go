package main

import "fmt"

func main() {
	fmt.Println("Structs in Golang")
	// no inheritance in golang; No super or parent

	brajeswar := User{"Brajeswar", "brajeswar@go.dev", true, 28}

	fmt.Println(brajeswar)
	fmt.Printf("Brajeswar details are: %v\n", brajeswar)
	fmt.Printf("Brajeswar details are: %+v\n", brajeswar)
	fmt.Printf("Name is %v and Email is %v.\n", brajeswar.Name, brajeswar.Email)

}

type User struct {
	Name   string
	Email  string
	Status bool
	Age    int
}
