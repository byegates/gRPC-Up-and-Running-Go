package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "productinfo/service/product/proto"
)

const (
	addr = "localhost:50081" // 71: java, 81: go
	tag  = "[Client]"
)

func main() {
	con, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatalf("%vFailed to connect: %v\n", tag, err)
	}

	// close connection with error handling
	defer func(con *grpc.ClientConn) {
		err := con.Close()
		if err != nil {
			log.Fatalf("%v[Close] [Error]: %v\n", tag, err)
		}
	}(con)

	c := pb.NewProductInfoClient(con)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	product, err := c.GetProduct(ctx, &pb.ProductID{Value: "9d6800bb-4321-44d1-a102-bfb4b301793a"})
	if err != nil {
		log.Printf("%v [R] [Error]: %v\n\n", tag, err)
	}
	log.Printf("%v [R] [Success] %v \n", tag, product)

	name := "Apple iPhone 14 Plus"
	desc := "Big and bigger."
	price := float32(899.0)

	r, err := c.AddProduct(ctx, &pb.Product{Name: name, Description: desc, Price: price})

	if err != nil {
		log.Fatalf("%v [Error] Fail to add product: %v\n\n", tag, err)
	}
	log.Printf("%v [C] [Success] %v \n", tag, r.Value)

	product, err = c.GetProduct(ctx, &pb.ProductID{Value: r.Value})
	if err != nil {
		log.Fatalf("%v [Error] Fail to read product: %v\n\n", tag, err)
	}
	log.Printf("%v [R] [Success] %v \n", tag, product)

}
