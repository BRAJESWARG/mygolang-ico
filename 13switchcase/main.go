package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	fmt.Println("Welcome Switch Case in Golang")

	rand.Seed(time.Now().UnixNano())
	diceNumber := rand.Intn(6) + 1
	fmt.Println("Vaalue of dice is ", diceNumber)

	switch diceNumber {
	case 1:
		fmt.Println("You can move to 1 spot")
	case 2:
		fmt.Println("You can move to 2 spot")
	case 3:
		fmt.Println("You can move to 3 spot")
	case 4:
		fmt.Println("You can move to 4 spot")
		fallthrough
	case 5:
		fmt.Println("You can move to 5 spot")
	case 6:
		fmt.Println("Dice value is 6 and you can open and roll dice again.")
	default:
		fmt.Println("What was that!")
	}

}
