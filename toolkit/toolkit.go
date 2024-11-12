package tool

import (
	"os"

	"github.com/bford/golang-x-crypto/ed25519"
	"github.com/bford/golang-x-crypto/ed25519/cosi"
)

func GenerateAggregationDataPerCommittee(pubKeys []ed25519.PublicKey) ([]byte, []byte) {
	cosigners := cosi.NewCosigners(pubKeys, nil)
	commits := []cosi.Commitment{}

	for i := 0; i < len(pubKeys); i++ {
		tempCommit, _, _ := cosi.Commit(nil)
		commits = append(commits, tempCommit)
	}

	aggregatePublicKey := cosigners.AggregatePublicKey()
	aggregateCommit := cosigners.AggregateCommit(commits)

	//fmt.Println("Aggregated Public Key: ", aggregatePublicKey)
	//fmt.Println("Aggregated Commit: ", aggregateCommit)

	return aggregateCommit, aggregatePublicKey
}

func GetEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}
	return value
}
