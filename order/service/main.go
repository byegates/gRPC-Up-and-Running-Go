package main

import (
	"context"
	pb "ecommerce/order/proto"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"net"
	"strings"
)

const (
	port           = ":50082"
	tag            = "[Server]"
	orderBatchSize = 3
)

var orderMap = make(map[string]*pb.Order)

type Server struct {
	pb.OrderManagementServer
	//pb.UnimplementedOrderManagementServer
	orderMap map[string]*pb.Order
}

func (s *Server) mustEmbedUnimplementedOrderManagementServer() {

}

func main() {
	initSampleData()
	lis, err := net.Listen("tcp", port)

	if err != nil {
		log.Fatalf("%v failed to listen %v\n\b", tag, err)
	}
	log.Printf("%v Listening on port %v\n\n", tag, port)

	s := grpc.NewServer()
	pb.RegisterOrderManagementServer(s, &Server{})
	// Register reflection service on gRPC server.
	// reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("%v failed to serve: %v\n\n", tag, err)
	}
}

// GetOrder Simple RPC
func (s *Server) GetOrder(_ context.Context, orderId *pb.OrderId) (*pb.Order, error) {
	tag0 := tag + " [R]"
	log.Printf("%v [Invoked]\n", tag0)
	defer log.Printf("%v [End]\n\n", tag0)

	ord, exists := orderMap[orderId.Id]
	if exists {
		return ord, status.New(codes.OK, "").Err()
	}

	return nil, status.Errorf(codes.NotFound, "Order does not exist. : ", orderId)
}

// AddOrder Simple RPC
func (s *Server) AddOrder(_ context.Context, req *pb.Order) (*pb.OrderId, error) {
	tag0 := tag + " [C]"
	log.Printf("%v [Invoked]\n", tag0)
	defer log.Printf("%v [End]\n\n", tag0)

	orderMap[req.Id] = req
	return &pb.OrderId{Id: req.Id}, nil
}

// SearchOrders Server-side Streaming RPC
func (s *Server) SearchOrders(req *pb.SearchRequest, stream pb.OrderManagement_SearchOrdersServer) error {
	tag0 := tag + " [SS]"
	log.Printf("%v [Invoked]\n", tag0)
	defer log.Printf("%v [End]\n\n", tag0)

	for key, ord := range orderMap {
		log.Printf("%v [ORDER] %v\n", tag0, ord)
		for _, itemName := range ord.Items {
			log.Printf("%v [ITEM]\t%v\n", tag0, itemName)
			if strings.Contains(itemName, req.S) {
				// Send the matching orders in a stream
				err := stream.Send(ord)
				if err != nil {
					return fmt.Errorf("error sending message to stream : %v", err)
				}
				log.Printf("%v [Found] %v\n", tag0, key)
				break
			}
		}
	}

	return nil
}

// UpdateOrders Client-side Streaming RPC
func (s *Server) UpdateOrders(stream pb.OrderManagement_UpdateOrdersServer) error {
	tag0 := tag + " [CS-UO]"
	log.Printf("%v [Invoked]\n", tag0)
	defer log.Printf("%v [End]\n\n", tag0)

	var orders []string
	for {
		order, err := stream.Recv()
		if err == io.EOF {
			// Finished reading the order stream.
			return stream.SendAndClose(&pb.UpdateOrdersRequest{Id: orders})
		}

		if err != nil {
			return err
		}
		// Update order
		orderMap[order.Id] = order

		log.Printf("%v Order ID : %s - Updated\n", tag0, order.Id)
		orders = append(orders, order.Id)
	}
}

// ProcessOrders Bi-directional Streaming RPC
func (s *Server) ProcessOrders(stream pb.OrderManagement_ProcessOrdersServer) error {
	tag0 := tag + " [BI]"
	log.Printf("%v [Invoked]\n", tag0)
	defer log.Printf("%v [End]\n\n", tag0)

	batchMarker := 1
	var shipmentMap = make(map[string]*pb.CombinedShipment)
	for {
		orderId, err := stream.Recv()
		if err == io.EOF {
			// Client has sent all the messages
			// Send remaining shipments
			log.Printf("%v [EOF]\n", tag0)
			for _, shipment := range shipmentMap {
				if err := stream.Send(shipment); err != nil {
					return err
				}
			}
			return nil
		}

		if err != nil {
			log.Println(err)
			return err
		}

		log.Printf("%v [Recv] %v\n", tag0, orderId)
		destination := orderMap[orderId.Id].Destination
		shipment, found := shipmentMap[destination]

		if !found {
			shipment = &pb.CombinedShipment{Id: fmt.Sprint(orderMap[orderId.Id].Destination)}
			shipmentMap[destination] = shipment
		}
		shipment.OrdersList = append(shipment.OrdersList, orderMap[orderId.Id])

		if batchMarker == orderBatchSize {
			for _, comb := range shipmentMap {
				log.Printf("%v [CMB Shipping] %20v -> %v\n", tag0, comb.Id, len(comb.OrdersList))
				if err := stream.Send(comb); err != nil {
					return err
				}
			}
			batchMarker = 0
			shipmentMap = make(map[string]*pb.CombinedShipment)
		} else {
			batchMarker++
		}
	}
}

func initSampleData() {
	orderMap["102"] = &pb.Order{Id: "102", Items: []string{"Google Pixel 3A", "Mac Book Pro"}, Destination: "Mountain View, CA", Price: 1800.00}
	orderMap["103"] = &pb.Order{Id: "103", Items: []string{"Apple Watch S4"}, Destination: "San Jose, CA", Price: 400.00}
	orderMap["104"] = &pb.Order{Id: "104", Items: []string{"Google Home Mini", "Google Nest Hub"}, Destination: "Mountain View, CA", Price: 400.00}
	orderMap["105"] = &pb.Order{Id: "105", Items: []string{"Amazon Echo"}, Destination: "San Jose, CA", Price: 30.00}
	orderMap["106"] = &pb.Order{Id: "106", Items: []string{"Amazon Echo", "Apple iPhone XS"}, Destination: "Mountain View, CA", Price: 300.00}
}
