package main

import "fmt"

func main() {
	fmt.Println("Welcome to Funtions in Golang")
	greeter()

	result := adder(3, 5)
	fmt.Println("Result is: ", result)

	proResult, myMessage := proAdder(3, 5, 6, 4, 8, 9)
	fmt.Println("Pro Result is: ", proResult)
	fmt.Println("Pro Message is: ", myMessage)

}

func greeter() {
	fmt.Println("Namastey from golang")
}

func adder(valOne int, valTwo int) int {
	return valOne + valTwo
}

func proAdder(values ...int) (int, string) {
	total := 0

	for _, val := range values {
		total += val
		fmt.Println(total)
	}
	return total, "Hi Pro result function"
}
