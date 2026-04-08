package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
)

func runClient() error {
	// Establish connection to server
	log.Printf("🔌 Connecting to gRPC server at localhost:50051...\n")
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	defer conn.Close()

	log.Printf("✅ Connected! Creating client stub...\n\n")

	// Create client stub (auto-generated code from proto)
	client := NewUserServiceClient(conn)

	// ========== TEST #1: CREATE USERS ==========
	log.Printf("========== TEST 1: CREATE USERS ==========\n")
	users := []struct {
		name  string
		email string
		age   int32
	}{
		{"Alice Johnson", "alice@example.com", 28},
		{"Bob Smith", "bob@example.com", 34},
		{"Carol White", "carol@example.com", 31},
	}

	for _, u := range users {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		resp, err := client.CreateUser(ctx, &CreateUserRequest{
			Name:  u.name,
			Email: u.email,
			Age:   u.age,
		})
		cancel()

		if err != nil {
			log.Printf("❌ Error creating user: %v\n", err)
			continue
		}

		log.Printf("   Created: ID=%d, Name=%s, Email=%s\n", resp.User.Id, resp.User.Name, resp.User.Email)
	}
	log.Printf("\n")

	// ========== TEST #2: GET SINGLE USER ==========
	log.Printf("========== TEST 2: GET SINGLE USER ==========\n")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	user, err := client.GetUser(ctx, &GetUserRequest{Id: 1})
	cancel()

	if err != nil {
		log.Printf("❌ Error getting user: %v\n", err)
	} else {
		log.Printf("   Found: %s (age %d)\n", user.Name, user.Age)
	}
	log.Printf("\n")

	// ========== TEST #3: LIST USERS (PAGINATION) ==========
	log.Printf("========== TEST 3: LIST USERS ==========\n")
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	listResp, err := client.ListUsers(ctx, &ListUsersRequest{
		Limit:  10,
		Offset: 0,
	})
	cancel()

	if err != nil {
		log.Printf("❌ Error listing users: %v\n", err)
	} else {
		log.Printf("   Total users: %d\n", listResp.Total)
		for _, u := range listResp.Users {
			log.Printf("   - %d: %s <%s> (age %d)\n", u.Id, u.Name, u.Email, u.Age)
		}
	}
	log.Printf("\n")

	// ========== TEST #4: UPDATE USER ==========
	log.Printf("========== TEST 4: UPDATE USER ==========\n")
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	updateResp, err := client.UpdateUser(ctx, &UpdateUserRequest{
		Id:    1,
		Name:  "Alice Johnson (Updated)",
		Email: "alice.updated@example.com",
		Age:   29,
	})
	cancel()

	if err != nil {
		log.Printf("❌ Error updating user: %v\n", err)
	} else {
		log.Printf("   Updated: %s -> %s\n", updateResp.User.Name, updateResp.User.Email)
	}
	log.Printf("\n")

	// ========== TEST #5: STREAM USERS ==========
	log.Printf("========== TEST 5: STREAM USERS (Server Streaming) ==========\n")
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*5)
	stream, err := client.StreamUsers(ctx, &ListUsersRequest{Limit: 100})
	cancel()

	if err != nil {
		log.Printf("❌ Error streaming users: %v\n", err)
	} else {
		count := 0
		for {
			user, err := stream.Recv()
			if err != nil {
				break
			}
			count++
			log.Printf("   Streamed user #%d: %s\n", count, user.Name)
		}
		log.Printf("   ✅ Received %d users via stream\n", count)
	}
	log.Printf("\n")

	// ========== TEST #6: DELETE USER ==========
	log.Printf("========== TEST 6: DELETE USER ==========\n")
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	deleteResp, err := client.DeleteUser(ctx, &DeleteUserRequest{Id: 2})
	cancel()

	if err != nil {
		log.Printf("❌ Error deleting user: %v\n", err)
	} else {
		log.Printf("   Deleted: %s\n", deleteResp.Message)
	}
	log.Printf("\n")

	// ========== VERIFY FINAL STATE ==========
	log.Printf("========== FINAL STATE ==========\n")
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	finalList, _ := client.ListUsers(ctx, &ListUsersRequest{Limit: 100})
	cancel()

	log.Printf("   Remaining users: %d\n", finalList.Total)
	for _, u := range finalList.Users {
		log.Printf("   - %d: %s\n", u.Id, u.Name)
	}

	return nil
}

func main() {
	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║        gRPC Client: Testing User Service                     ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝\n")

	if err := runClient(); err != nil {
		log.Fatalf("Client error: %v", err)
	}
}
