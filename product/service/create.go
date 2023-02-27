package main

import (
	"context"
	pb "ecommerce/product/proto"
	"log"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *server) AddProduct(ctx context.Context, in *pb.Product) (*pb.ProductID, error) {
	tag0 := tag + " [C]"
	log.Printf("%v [Invoked]\n", tag0)

	out, err := uuid.NewUUID()

	if err != nil {
		return nil, status.Errorf(codes.Internal, "[Error] Generating Product ID", err)
	}

	in.Id = out.String()
	s.productMap[in.Id] = in

	return &pb.ProductID{Value: in.Id}, status.New(codes.OK, "").Err()
}
