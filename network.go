package main

import (
	"context"
	"encoding/json"
	pb "new-interface/bcinterface"
	"sync"
	"time"

	"fmt"
	"log"
	"net"
	tool "new-interface/toolkit"

	"github.com/bford/golang-x-crypto/ed25519"
	"github.com/gomodule/redigo/redis"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
)

type Server struct {
	pb.UnimplementedBlockchainInterfaceServer
}

type Cache struct {
	connRedis redis.Conn
	connMongo *mongo.Client
}

type nodeAccountEntity struct {
	Address   string `json:"address"`
	Pubkey    string `json:"pubkey"`
	Signature string `json:"signature(sK(Address))"`
}

type vrfValue struct {
	Val    string `json:"val"`
	Proof  []byte `json:"pi"`
	PubKey []byte `json:"pk"`
}

type CommitteeNodeInfo struct {
	Round     int32    `json:"round"`
	Address   string   `json:"address"`
	VrfPubKey []byte   `json:"vrfpubkey"`
	VrfResult vrfValue `json:"vrfresult"`
}

type CommitteeInfo struct {
	AggregateCommit []byte              `json:"aggcommit"`
	AggregatePubKey []byte              `json:"aggpubkey"`
	CommitteeList   []CommitteeNodeInfo `json:"committeelist"`
	PrimaryNodeInfo string
}

var singleCache *Cache
var initOnce sync.Once

var portNumber = tool.GetEnv("GRPC_PORT", ":50051")
var mongoUrl = tool.GetEnv("MONGO_ADDR", "localhost")
var redisUrl = tool.GetEnv("REDIS_ADDR", "localshot")
var redisPort = tool.GetEnv("REDIS_PORT", ":6379")

var sendCommitteeList []CommitteeNodeInfo
var recvPartPubKey []ed25519.PublicKey

func StartGrpcServer() {
	lis, err := net.Listen("tcp", portNumber)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterBlockchainInterfaceServer(s, &Server{})
	fmt.Println("*************************************************************")
	fmt.Println("*                                                           *")
	fmt.Println("*                Running Interface container                *")
	fmt.Println("*                                                           *")
	fmt.Println("*************************************************************")

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func GetConnectorCache() *Cache {
	initOnce.Do(func() {
		//redis init
		singleCache = &Cache{}
		//singleCache.connRedis, _ = redis.DialURL("redis://" + redisUrl + ":6379") //("redis://redis:6379")
		singleCache.connRedis, _ = redis.DialURL("redis://" + redisUrl + redisPort) //("redis://redis:6379")

		fmt.Println("GetConnectorCache : Redis Connection Made")
		//mongo init
		credential := options.Credential{
			Username: "root",
			Password: "root",
		}
		clientOptions := options.Client().ApplyURI("mongodb://" + mongoUrl + ":27017").SetAuth(credential)
		//("mongodb://mongo:27017").SetAuth(credential)
		singleCache.connMongo, _ = mongo.Connect(context.TODO(), clientOptions)

		//Check the connection
		err := singleCache.connMongo.Ping(context.TODO(), nil)
		log.Println(err)
		fmt.Println("GetConnectorCache : MongoDB Connection Made")

	})
	return singleCache
}

func (s *Server) EnrollNodeInfo(ctx context.Context, in *pb.NodeData) (*pb.EnrollAccountResponse, error) {
	fmt.Println("Start Node Enrollment")

	conn := GetConnectorCache()
	inPartKey := nodeAccountEntity{in.Address, string(in.Pubkey), string(in.Signature)}

	// 중복 값 확인
	filter := bson.M{"address": in.Address, "pk": in.Pubkey, "sig": in.Signature}
	num, err := conn.connMongo.Database("nodeData").Collection("nodeData").CountDocuments(ctx, filter)
	if err != nil {
		fmt.Println(err)
	}

	if num == 0 {
		_, err := conn.connMongo.Database("nodeData").Collection("nodeData").InsertOne(ctx, inPartKey)
		if err != nil {
			fmt.Println(err)
			return &pb.EnrollAccountResponse{Code: 500}, nil
		}
	}

	fmt.Println("successfully send 200 response")
	return &pb.EnrollAccountResponse{Code: 200}, nil
}

func PublishMessageToRedis(channelName string, message []byte) {
	c, _ := redis.DialURL("redis://" + redisUrl + redisPort)
	if c == nil {
		fmt.Println("err")
	}
	fmt.Println("PublishMsg: ", string(message))

	c.Do("PUBLISH", channelName, message)
	fmt.Println("COMMITTEE SELECTION END | ", time.Now().UnixNano()/int64(time.Millisecond))

}

func (s *Server) SetupCommittee(ctx context.Context, in *pb.SetupCommitteeRequest) (*pb.SetupCommitteeResponse, error) {
	fmt.Println("Get Setup Committee Request")
	var recvCommitteeInfo CommitteeNodeInfo

	recvCommitteeInfo.Round = in.Round
	recvCommitteeInfo.Address = in.Nodeip
	recvCommitteeInfo.VrfPubKey = in.Vrfpubkey
	recvCommitteeInfo.VrfResult.Val = in.VrfResult.Val
	recvCommitteeInfo.VrfResult.Proof = in.VrfResult.Proof
	recvCommitteeInfo.VrfResult.PubKey = in.VrfResult.Pubkey

	sendCommitteeList = append(sendCommitteeList, recvCommitteeInfo)
	recvPartPubKey = append(recvPartPubKey, recvCommitteeInfo.VrfPubKey)

	if len(sendCommitteeList) == 4 {
		var sendCommitteeInfo = CommitteeInfo{}
		sendCommitteeInfo.CommitteeList = sendCommitteeList
		sendCommitteeInfo.PrimaryNodeInfo = "PRIMARY_NODE_INFO" //차후 이에대해 정해지면 추가

		aggCommit, aggPubKey := tool.GenerateAggregationDataPerCommittee(recvPartPubKey)
		sendCommitteeInfo.AggregateCommit = aggCommit
		sendCommitteeInfo.AggregatePubKey = aggPubKey

		fmt.Println("Selected Committee Info: ", sendCommitteeInfo)

		sendCommitteeInfoBytes, error := json.Marshal(sendCommitteeInfo)
		if error != nil {
			log.Fatal(error)
		}
		PublishMessageToRedis("CommitteeList", sendCommitteeInfoBytes)
		sendCommitteeList = []CommitteeNodeInfo{}
	}

	//redis로 전송하고 나면 200 반환
	return &pb.SetupCommitteeResponse{Code: 200}, nil
}

func (s *Server) LeaveRequest(ctx context.Context, in *pb.NodeData) (*pb.Empty, error) {
	conn := GetConnectorCache()
	collection := conn.connMongo.Database("partKeyStore").Collection("partKeyStore")

	filter := bson.M{"address": in.Address, "pk": in.Pubkey, "sig": in.Signature}

	delResult, err := collection.DeleteOne(context.TODO(), filter)
	if err != nil {
		log.Println("can't delete leaving node data")
	}

	fmt.Printf("Deleted %v document", delResult.DeletedCount)

	return &pb.Empty{}, nil
}
