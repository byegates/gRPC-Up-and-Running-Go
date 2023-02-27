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

var orderMap = make(map[string]pb.Order)

//var m = sync.Map{}

type Server struct {
	pb.OrderManagementServer
	//pb.UnimplementedOrderManagementServer
	orderMap map[string]*pb.Order
	//m        sync.Map
}

func (s *Server) mustEmbedUnimplementedOrderManagementServer() {

}

func main() {
	initSampleData()
	lis, err := net.Listen("tcp", port)

	if err != nil {
		log.Fatalf("%v failed to listen %v\n\b", tag, err)
	}
	log.Printf("%v Listening on port :%v\n\n", tag, port)

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
	tag0 := tag + " [C]"
	log.Printf("%v [Invoked]\n\n", tag0)
	ord, exists := orderMap[orderId.Id]
	if exists {
		return &ord, status.New(codes.OK, "").Err()
	}

	return nil, status.Errorf(codes.NotFound, "Order does not exist. : ", orderId)
}

// AddOrder Simple RPC
func (s *Server) AddOrder(_ context.Context, req *pb.Order) (*pb.OrderId, error) {
	tag0 := tag + " [C]"
	log.Printf("%v [Invoked] [ID] %v\n\n", tag0, req.Id)
	orderMap[req.Id] = *req
	//m.Store(req.Id, *req)
	return &pb.OrderId{Id: req.Id}, nil
}

// SearchOrders Server-side Streaming RPC
func (s *Server) SearchOrders(req *pb.SearchRequest, stream pb.OrderManagement_SearchOrdersServer) error {
	tag0 := tag + " [SS]"
	log.Printf("%v [Invoked]\n\n", tag0)

	for key, ord := range orderMap {
		log.Print(key, &ord)
		for _, itemStr := range ord.Items {
			log.Print(itemStr)
			if strings.Contains(itemStr, req.S) {
				// Send the matching orders in a stream
				err := stream.Send(&ord)
				if err != nil {
					return fmt.Errorf("error sending message to stream : %v", err)
				}
				log.Print("Matching Order Found : " + key)
				break
			}
		}
	}

	return nil
}

// UpdateOrders Client-side Streaming RPC
func (s *Server) UpdateOrders(stream pb.OrderManagement_UpdateOrdersServer) error {
	tag0 := tag + "[UO] [CS]"
	log.Printf("%v [Invoked]\n\n", tag0)

	var orders []string
	for {
		order, err := stream.Recv()
		if err == io.EOF {
			// Finished reading the order stream.
			return stream.SendAndClose(&pb.OrderIdList{Ids: orders})
		}

		if err != nil {
			return err
		}
		// Update order
		orderMap[order.Id] = *order

		log.Printf("%v Order ID : %s - Updated\n\n", order.Id, tag0)
		orders = append(orders, order.Id)
	}
}

// ProcessOrders Bi-directional Streaming RPC
func (s *Server) ProcessOrders(stream pb.OrderManagement_ProcessOrdersServer) error {

	batchMarker := 1
	var combinedShipmentMap = make(map[string]pb.CombinedShipment)
	for {
		orderId, err := stream.Recv()
		log.Printf("Reading Proc order : %s", orderId)
		if err == io.EOF {
			// Client has sent all the messages
			// Send remaining shipments
			log.Printf("EOF : %s", orderId)
			for _, shipment := range combinedShipmentMap {
				if err := stream.Send(&shipment); err != nil {
					return err
				}
			}
			return nil
		}
		if err != nil {
			log.Println(err)
			return err
		}

		destination := orderMap[orderId.Id].Destination
		shipment, found := combinedShipmentMap[destination]

		if found {
			ord := orderMap[orderId.Id]
			shipment.OrdersList = append(shipment.OrdersList, &ord)
			combinedShipmentMap[destination] = shipment
		} else {
			comShip := pb.CombinedShipment{Id: "cmb - " + (orderMap[orderId.Id].Destination), Status: "Processed!"}
			ord := orderMap[orderId.Id]
			comShip.OrdersList = append(shipment.OrdersList, &ord)
			combinedShipmentMap[destination] = comShip
			log.Print(len(comShip.OrdersList), comShip.GetId())
		}

		if batchMarker == orderBatchSize {
			for _, comb := range combinedShipmentMap {
				log.Printf("Shipping : %v -> %v", comb.Id, len(comb.OrdersList))
				if err := stream.Send(&comb); err != nil {
					return err
				}
			}
			batchMarker = 0
			combinedShipmentMap = make(map[string]pb.CombinedShipment)
		} else {
			batchMarker++
		}
	}
}

func initSampleData() {
	//m.Store("102", pb.Order{Id: "102", Items: []string{"Google Pixel 3A", "Mac Book Pro"}, Destination: "Mountain View, CA", Price: 1800.00})
	//m.Store("103", pb.Order{Id: "103", Items: []string{"Apple Watch S4"}, Destination: "San Jose, CA", Price: 400.00})
	//m.Store("104", pb.Order{Id: "104", Items: []string{"Google Home Mini", "Google Nest Hub"}, Destination: "Mountain View, CA", Price: 400.00})
	//m.Store("105", pb.Order{Id: "105", Items: []string{"Amazon Echo"}, Destination: "San Jose, CA", Price: 30.00})
	//m.Store("106", pb.Order{Id: "106", Items: []string{"Amazon Echo", "Apple iPhone XS"}, Destination: "Mountain View, CA", Price: 300.00})

	orderMap["102"] = pb.Order{Id: "102", Items: []string{"Google Pixel 3A", "Mac Book Pro"}, Destination: "Mountain View, CA", Price: 1800.00}
	orderMap["103"] = pb.Order{Id: "103", Items: []string{"Apple Watch S4"}, Destination: "San Jose, CA", Price: 400.00}
	orderMap["104"] = pb.Order{Id: "104", Items: []string{"Google Home Mini", "Google Nest Hub"}, Destination: "Mountain View, CA", Price: 400.00}
	orderMap["105"] = pb.Order{Id: "105", Items: []string{"Amazon Echo"}, Destination: "San Jose, CA", Price: 30.00}
	orderMap["106"] = pb.Order{Id: "106", Items: []string{"Amazon Echo", "Apple iPhone XS"}, Destination: "Mountain View, CA", Price: 300.00}
}
