package main

import (
	"fmt"
	"sync"
)

func main() {
	fmt.Println("Welcome to channels in Golang")

	myCh := make(chan int, 2)

	wg := &sync.WaitGroup{}

	// fmt.Println(<-myCh)
	// myCh <- 5

	wg.Add(2)

	// RECEIVE ONLY
	go func(ch <-chan int, wg *sync.WaitGroup) {
		val, isChannelOpen := <-myCh

		fmt.Println(isChannelOpen)
		fmt.Println(val)

		// fmt.Println(<-myCh)
		// fmt.Println(<-myCh)
		wg.Done()
	}(myCh, wg)

	// SEND ONLY
	go func(ch chan<- int, wg *sync.WaitGroup) {
		myCh <- 5
		myCh <- 6
		wg.Done()
	}(myCh, wg)

	wg.Wait()

}
