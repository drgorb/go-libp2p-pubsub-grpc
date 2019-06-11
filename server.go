package libpubsubgrpc

import (
	"context"
	"errors"
	"net"

	"github.com/RTradeLtd/go-libp2p-pubsub-grpc/pb"
	ps "github.com/libp2p/go-libp2p-pubsub"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Server is used to run the libp2p pubsub grpc server
// it enables communicating pubsub messages over grpc
type Server struct {
	pb pb.PubSubServiceServer
	ps *ps.PubSub
}

// NewServer is used to intiialize a pubsub grpc server and run it
func NewServer(ctx context.Context, pubsub *ps.PubSub, logger *zap.SugaredLogger, insecure bool, protocol, url string) error {
	lis, err := net.Listen(protocol, url)
	if err != nil {
		return err
	}
	var serverOpts []grpc.ServerOption
	if !insecure {
		serverOpts, err = options("", "", "", logger)
		if err != nil {
			return err
		}
	}
	srv := &Server{ps: pubsub}
	gServer := grpc.NewServer(serverOpts...)
	pb.RegisterPubSubServiceServer(gServer, srv)
	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Info("shutting server down")
				gServer.GracefulStop()
				return
			}
		}
	}()
	return gServer.Serve(lis)
}

// GetTopics is used to return a list of all known topics the pubsub instance is subscribed to.
func (s *Server) GetTopics(ctx context.Context, req *pb.Empty) (*pb.TopicsResponse, error) {
	return &pb.TopicsResponse{Names: s.ps.GetTopics()}, nil
}

// ListPeers is used to return a list of peers subscribed to a given topic or topics
func (s *Server) ListPeers(ctx context.Context, req *pb.ListPeersRequest) (*pb.ListPeersResponse, error) {
	var peers []*pb.ListPeersResponse_Peer
	for _, topic := range req.GetTopics() {
		pids := s.ps.ListPeers(topic)
		for _, pid := range pids {
			peers = append(peers, &pb.ListPeersResponse_Peer{Topic: topic, PeerID: pid.String()})
		}
	}
	return &pb.ListPeersResponse{Peers: peers}, nil
}

// Subscribe is used to subscribe to a topic and receive messages
func (s *Server) Subscribe(req *pb.SubscribeRequest, stream pb.PubSubService_SubscribeServer) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub, err := s.ps.Subscribe(req.GetTopic())
	if err != nil {
		return err
	}
	for {
		proto2Msg, err := sub.Next(ctx)
		if err != nil {
			return err
		}
		proto3Msg := &pb.PubSubMessageResponse{
			From:      []byte(proto2Msg.GetFrom().String()),
			Data:      proto2Msg.GetData(),
			Seqno:     proto2Msg.GetSeqno(),
			TopicIDs:  proto2Msg.GetTopicIDs(),
			Signature: proto2Msg.GetSignature(),
			Key:       proto2Msg.GetKey(),
		}
		if err := stream.Send(proto3Msg); err != nil {
			return err
		}
	}
}

// Publish is used to send a stream of messages to a pubsub topic.
func (s *Server) Publish(stream pb.PubSubService_PublishServer) error {
	return errors.New("coming soon")
}
