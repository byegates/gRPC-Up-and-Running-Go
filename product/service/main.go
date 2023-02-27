package main

import (
	"log"
	"net"

	pb "productinfo/service/product/proto"

	"google.golang.org/grpc"
)

const (
	port = ":50081"
	tag  = "[Server]"
)

type server struct {
	pb.ProductInfoServer
	productMap map[string]*pb.Product
}

func main() {
	lis, err := net.Listen("tcp", port)

	if err != nil {
		log.Fatalf("%v failed to listen %v\n\b", tag, err)
	}
	log.Printf("%v Listening on port :%v\n\n", tag, port)

	s := grpc.NewServer()
	pb.RegisterProductInfoServer(s, &server{productMap: make(map[string]*pb.Product)})

	if err := s.Serve(lis); err != nil {
		log.Fatalf("%v failed to serve: %v\n\n", tag, err)
	}
}
