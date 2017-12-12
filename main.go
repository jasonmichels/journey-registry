package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	cf "github.com/jasonmichels/go-journey-server-utils/config"
	"github.com/jasonmichels/journey-registry/journey"
	"github.com/jasonmichels/journey-registry/server"
)

func main() {
	port := cf.Getenv("PORT", ":80")
	awsRegion := cf.Getenv("AWS_REGION", "us-east-1")
	bucket := os.Getenv("AWS_BUCKET")

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	awsConfig := aws.Config{Region: aws.String(awsRegion)}
	sess, err := session.NewSession(&awsConfig)
	if err != nil {
		log.Fatalf("Error creating AWS session %v", err.Error())
	}

	cache := make(map[string]*server.CacheV)

	s := grpc.NewServer()
	journey.RegisterExplorerServer(s, &server.Explorer{VersionCache: cache, AWS: sess, Bucket: bucket})

	// Register reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve journey server: %v", err)
	}
}
