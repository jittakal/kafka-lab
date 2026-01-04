package kafka

import (
	"crypto/sha256"
	"crypto/sha512"
	"hash"

	"github.com/IBM/sarama"
	"github.com/xdg-go/scram"
)

// XDGSCRAMClient implements sarama.SCRAMClient for SCRAM authentication.
type XDGSCRAMClient struct {
	*scram.Client
	*scram.ClientConversation
	scram.HashGeneratorFcn
}

// Begin starts the SCRAM authentication process.
func (x *XDGSCRAMClient) Begin(userName, password, authzID string) (err error) {
	x.Client, err = x.HashGeneratorFcn.NewClient(userName, password, authzID)
	if err != nil {
		return err
	}
	x.ClientConversation = x.Client.NewConversation()
	return nil
}

// Step performs a step in the SCRAM authentication.
func (x *XDGSCRAMClient) Step(challenge string) (response string, err error) {
	return x.ClientConversation.Step(challenge)
}

// Done indicates if authentication is complete.
func (x *XDGSCRAMClient) Done() bool {
	return x.ClientConversation.Done()
}

// SHA256 returns a SHA256 hash generator.
func SHA256() scram.HashGeneratorFcn {
	return func() hash.Hash { return sha256.New() }
}

// SHA512 returns a SHA512 hash generator.
func SHA512() scram.HashGeneratorFcn {
	return func() hash.Hash { return sha512.New() }
}

// Ensure XDGSCRAMClient implements sarama.SCRAMClient.
var _ sarama.SCRAMClient = (*XDGSCRAMClient)(nil)
