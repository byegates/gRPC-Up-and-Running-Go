package main

import (
	"context"
	pb "ecommerce/order/proto"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	"time"
)

const (
	addr = "localhost:50082" // 71: java, 81: go
	tag  = "[Client]"
)

func main() {
	// Setting up a connection to the server.
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(orderUnaryClientInterceptor),
		grpc.WithStreamInterceptor(clientStreamInterceptor))

	if err != nil {
		log.Fatalf("%vFailed to connect: %v\n", tag, err)
	}

	defer func(conn *grpc.ClientConn) {
		err := conn.Close()
		if err != nil {
			log.Fatalf("%v[Close] [Error]: %v\n", tag, err)
		}
	}(conn)

	c := pb.NewOrderManagementClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	id := addOrder(ctx, c)
	getOrder(ctx, c, id)
	searchOrders(ctx, c, "Google")
	updateOrders(ctx, c)
	processOrders(ctx, c)
}

// Add Order
func addOrder(ctx context.Context, c pb.OrderManagementClient) string {
	tag0 := tag + " [C]"
	log.Printf("%v [Invoked]\n", tag0)
	defer log.Printf("%v [End]\n\n", tag0)

	ord := pb.Order{Id: "101", Items: []string{"iPhone XS", "Mac Book Pro"}, Destination: "San Jose, CA", Price: 2299.00}
	log.Printf("%v [Creating] %v\n", tag0, &ord)

	_, err := c.AddOrder(ctx, &ord)
	if err != nil {
		log.Printf("%v[Error] %v\n\n", tag0, err)
		return ""
	}
	log.Printf("%v [Success]\n", tag0)
	return ord.Id
}

// Get Order
func getOrder(ctx context.Context, c pb.OrderManagementClient, id string) {
	tag0 := tag + " [R]"
	log.Printf("%v [Invoked]\n", tag0)
	defer log.Printf("%v [End]\n\n", tag0)

	ord, err := c.GetOrder(ctx, &pb.OrderId{Id: id})

	if err != nil {
		log.Printf("%v [Error] %v\n\n", tag0, err)
	}

	log.Printf("%v [Success] %v\n", tag0, ord)
}

// Search Order : Server streaming scenario
func searchOrders(ctx context.Context, c pb.OrderManagementClient, s string) {
	tag0 := tag + " [SS]"
	log.Printf("%v [Invoked]\n", tag0)
	defer log.Printf("%v [End]\n\n", tag0)

	stream, _ := c.SearchOrders(ctx, &pb.SearchRequest{S: s})
	for {
		order, err := stream.Recv()

		if err == io.EOF {
			log.Printf("%v [EOF]\n", tag0)
			break
		}

		if err != nil {
			log.Printf("%v [Error] %v\n\n", tag0, err)
			continue
		}

		log.Printf("%v [Recv] %v\n", tag0, order)
	}

}

// Update Orders : Client streaming scenario
func updateOrders(ctx context.Context, c pb.OrderManagementClient) {
	tag0 := tag + " [CS]"
	log.Printf("%v [Invoked]\n", tag0)
	defer log.Printf("%v [End]\n\n", tag0)

	orders := []*pb.Order{
		{Id: "102", Items: []string{"Google Pixel 3A", "Google Pixel Book"}, Destination: "Mountain View, CA", Price: 1100.00},
		{Id: "103", Items: []string{"Apple Watch S4", "Mac Book Pro", "iPad Pro"}, Destination: "San Jose, CA", Price: 2800.00},
		{Id: "104", Items: []string{"Google Home Mini", "Google Nest Hub", "iPad Mini"}, Destination: "Mountain View, CA", Price: 2200.00},
	}

	stream, err := c.UpdateOrders(ctx)

	if err != nil {
		log.Fatalf("%v.UpdateOrders(_) = _, %v", c, err)
	}

	for _, ord := range orders {
		if err := stream.Send(ord); err != nil {
			log.Fatalf("%v %v.Send(%v) = %v", tag0, stream, &ord, err)
		}
	}

	ids, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("%v %v.CloseAndRecv() got error %v, want %v\n\n", tag0, stream, err, nil)
	}
	log.Printf("%v [Success] %s\n", tag0, ids)
}

// =========================================
// Process Order : Bi-di streaming scenario
func processOrders(ctx context.Context, client pb.OrderManagementClient) {
	tag0 := tag + " [BI]"
	log.Printf("%v [Invoked]\n", tag0)
	defer log.Printf("%v [End]\n\n", tag0)

	reqs := []*pb.OrderId{
		{Id: "102"},
		{Id: "103"},
		{Id: "104"},
		{Id: "101"},
	}

	stream, err := client.ProcessOrders(ctx)

	if err != nil {
		log.Fatalf("%v %v.ProcessOrders(_) = _, %v", tag0, client, err)
	}

	c := make(chan struct{})
	go asyncClientBidirectionalRPC(stream, c)

	for _, req := range reqs {
		if err := stream.Send(req); err != nil {
			log.Fatalf("%v %v.Send(%v) = %v", tag0, client, req.Id, err)
		}
	}

	if err := stream.CloseSend(); err != nil {
		log.Fatal(err)
	}

	<-c
}

func asyncClientBidirectionalRPC(streamProcOrder pb.OrderManagement_ProcessOrdersClient, c chan<- struct{}) {
	tag0 := tag + " [BI]"
	for {
		combinedShipment, err := streamProcOrder.Recv()
		if err == io.EOF {
			log.Printf("%v [%v]\n", tag0, err)
			break
		}
		if err != nil {
			log.Printf("%v [Error] %v\n", tag0, err)
			break
		}
		msg := fmt.Sprintf("%v\n\tCombined shipment :\n", tag0)
		for _, ord := range combinedShipment.OrdersList {
			msg += fmt.Sprintf("\t\t%v\n", ord)
		}
		log.Printf(msg)
	}
	c <- struct{}{}
}

func orderUnaryClientInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	// Pre-processor phase
	log.Printf("%v [Unary] %v\n", tag, method)

	// Invoking the remote method
	err := invoker(ctx, method, req, reply, cc, opts...)

	// Post-processor phase
	//log.Printf("%v [Resp] %v\n", tag, reply)

	return err
}

func clientStreamInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {

	log.Printf("%v [Stream Interceptor] %v\n", tag, method)
	s, err := streamer(ctx, desc, cc, method, opts...)
	if err != nil {
		return nil, err
	}
	return newWrappedStream(s), nil
}

type wrappedStream struct {
	grpc.ClientStream
}

func (w *wrappedStream) RecvMsg(m interface{}) error {
	log.Printf("%v [Stream Interceptor] [Recv] [Type: %T] [%v] [at %v]", tag, m, &m, time.Now().Format(time.RFC3339))
	return w.ClientStream.RecvMsg(m)
}

func (w *wrappedStream) SendMsg(m interface{}) error {
	log.Printf("%v [Stream Interceptor] [Send] [Type: %T] [%v] [at %v]", tag, m, &m, time.Now().Format(time.RFC3339))
	return w.ClientStream.SendMsg(m)
}

func newWrappedStream(s grpc.ClientStream) grpc.ClientStream {
	return &wrappedStream{s}
}
