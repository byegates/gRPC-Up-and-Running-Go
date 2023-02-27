package main

import (
	"context"
	"log"
	pb "productinfo/service/product/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *server) GetProduct(ctx context.Context, in *pb.ProductID) (*pb.Product, error) {
	tag0 := tag + " [R]"
	log.Printf("%v [Invoked]\n\n", tag0)
	value, exists := s.productMap[in.Value]

	if exists {
		return value, status.New(codes.OK, "").Err()
	}

	return nil, status.Errorf(codes.NotFound, "Product does not exist", in.Value)
}
