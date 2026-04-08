package main

import (
	"context"
	pb "grpc-go-demo/proto"
	"log"
	"net"

	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedUserServiceServer
}

func (s *server) GetUser(ctx context.Context, req *pb.UserRequest) (*pb.UserResponse, error) {
	log.Println("Request received for ID:", req.Id)

	return &pb.UserResponse{
		Id:    req.Id,
		Name:  "Alice",
		Email: "alice@example.com",
	}, nil
}

func main() {
	lis, _ := net.Listen("tcp", ":50051")
	grpcServer := grpc.NewServer()

	pb.RegisterUserServiceServer(grpcServer, &server{})

	log.Println("Server running on :50051")
	grpcServer.Serve(lis)
}
