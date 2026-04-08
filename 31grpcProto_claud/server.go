package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"google.golang.org/grpc"
)

// ========== IN-MEMORY DATA STORAGE ==========
type UserStore struct {
	mu    sync.RWMutex
	users map[int32]*User
	nextID int32
}

var store = &UserStore{
	users: make(map[int32]*User),
	nextID: 1,
}

// ========== USER SERVICE IMPLEMENTATION ==========
type UserServiceServer struct {
	UnimplementedUserServiceServer
}

// CreateUser - RPC Handler #1
func (s *UserServiceServer) CreateUser(ctx context.Context, req *CreateUserRequest) (*CreateUserResponse, error) {
	log.Printf("📝 CreateUser RPC called with: %s (%s), age %d\n", req.Name, req.Email, req.Age)

	// Validate input
	if req.Name == "" || req.Email == "" {
		return &CreateUserResponse{
			Success: false,
			Message: "Name and email are required",
		}, nil
	}

	// Store user
	store.mu.Lock()
	defer store.mu.Unlock()

	user := &User{
		Id:       store.nextID,
		Name:     req.Name,
		Email:    req.Email,
		Age:      req.Age,
		IsActive: true,
	}

	store.users[store.nextID] = user
	store.nextID++

	log.Printf("✅ User created with ID: %d\n", user.Id)

	return &CreateUserResponse{
		Success: true,
		Message: fmt.Sprintf("User %s created successfully", req.Name),
		User:    user,
	}, nil
}

// GetUser - RPC Handler #2
func (s *UserServiceServer) GetUser(ctx context.Context, req *GetUserRequest) (*User, error) {
	log.Printf("🔍 GetUser RPC called with ID: %d\n", req.Id)

	store.mu.RLock()
	defer store.mu.RUnlock()

	user, exists := store.users[req.Id]
	if !exists {
		return nil, fmt.Errorf("user with ID %d not found", req.Id)
	}

	log.Printf("✅ Found user: %s\n", user.Name)
	return user, nil
}

// ListUsers - RPC Handler #3 (Returns multiple messages)
func (s *UserServiceServer) ListUsers(ctx context.Context, req *ListUsersRequest) (*ListUsersResponse, error) {
	log.Printf("📋 ListUsers RPC called (limit: %d, offset: %d)\n", req.Limit, req.Offset)

	store.mu.RLock()
	defer store.mu.RUnlock()

	// Simple pagination
	offset := int(req.Offset)
	limit := int(req.Limit)
	if limit == 0 {
		limit = 10
	}

	var users []*User
	count := 0
	skipped := 0

	for _, user := range store.users {
		if skipped >= offset {
			if count >= limit {
				break
			}
			users = append(users, user)
			count++
		} else {
			skipped++
		}
	}

	log.Printf("✅ Returning %d users out of %d total\n", len(users), len(store.users))

	return &ListUsersResponse{
		Users: users,
		Total: int32(len(store.users)),
	}, nil
}

// UpdateUser - RPC Handler #4
func (s *UserServiceServer) UpdateUser(ctx context.Context, req *UpdateUserRequest) (*UpdateUserResponse, error) {
	log.Printf("✏️  UpdateUser RPC called for ID: %d\n", req.Id)

	store.mu.Lock()
	defer store.mu.Unlock()

	user, exists := store.users[req.Id]
	if !exists {
		return &UpdateUserResponse{
			Success: false,
		}, fmt.Errorf("user with ID %d not found", req.Id)
	}

	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Age > 0 {
		user.Age = req.Age
	}

	log.Printf("✅ User updated: %s\n", user.Name)

	return &UpdateUserResponse{
		Success: true,
		User:    user,
	}, nil
}

// DeleteUser - RPC Handler #5
func (s *UserServiceServer) DeleteUser(ctx context.Context, req *DeleteUserRequest) (*DeleteUserResponse, error) {
	log.Printf("🗑️  DeleteUser RPC called for ID: %d\n", req.Id)

	store.mu.Lock()
	defer store.mu.Unlock()

	_, exists := store.users[req.Id]
	if !exists {
		return &DeleteUserResponse{
			Success: false,
			Message: "User not found",
		}, nil
	}

	delete(store.users, req.Id)
	log.Printf("✅ User deleted\n")

	return &DeleteUserResponse{
		Success: true,
		Message: "User deleted successfully",
	}, nil
}

// StreamUsers - RPC Handler #6 (Server Streaming)
func (s *UserServiceServer) StreamUsers(req *ListUsersRequest, stream grpc.ServerStream) error {
	log.Printf("🌊 StreamUsers RPC called (streaming mode)\n")

	store.mu.RLock()
	defer store.mu.RUnlock()

	count := 0
	for _, user := range store.users {
		if count >= int(req.Limit) && req.Limit > 0 {
			break
		}

		// Send each user one by one
		if err := stream.SendMsg(user); err != nil {
			log.Printf("❌ Stream error: %v\n", err)
			return err
		}

		log.Printf("   📤 Streamed user: %s\n", user.Name)
		count++
	}

	log.Printf("✅ Stream completed (%d users sent)\n", count)
	return nil
}

// ========== SERVER STARTUP ==========
func startServer(port string) error {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %v", port, err)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()
	RegisterUserServiceServer(grpcServer, &UserServiceServer{})

	log.Printf("🚀 gRPC Server listening on port %s\n", port)
	log.Printf("📡 gRPC uses HTTP/2 with Protocol Buffers (binary format)\n\n")

	return grpcServer.Serve(listener)
}

func main() {
	if err := startServer("50051"); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
