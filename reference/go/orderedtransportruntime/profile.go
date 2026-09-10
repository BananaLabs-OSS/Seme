package orderedtransportruntime

import (
	"crypto/sha256"
	"fmt"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/orderedtransportinstance"
)

type Operation struct {
	Identity, Capability string
	Sequence             uint64
}

type Profile struct {
	CodecIdentity, DigestIdentity          string
	MaximumPayloadBytes, MaximumFrameBytes uint64
	Receive, Send                          Operation
	authentication                         [sha256.Size]byte
}

func AuthenticatedProfile(contract contractcatalog.Contract) (Profile, error) {
	a, err := orderedtransportinstance.Authenticate(contract)
	if err != nil {
		return Profile{}, err
	}
	p := Profile{CodecIdentity: a.CodecIdentity, DigestIdentity: a.DigestIdentity, MaximumPayloadBytes: a.MaximumPayloadBytes, MaximumFrameBytes: a.MaximumFrameBytes, Receive: Operation{Identity: a.ReceiveIdentity, Capability: a.ReceiveCapability, Sequence: 0}, Send: Operation{Identity: a.SendIdentity, Capability: a.SendCapability, Sequence: 1}}
	p.authentication = profileDigest(p)
	return p, nil
}

func validProfile(p Profile) bool {
	return p.authentication == profileDigest(p) && p.CodecIdentity == orderedtransportinstance.CodecIdentity && p.DigestIdentity == orderedtransportinstance.DigestIdentity && p.MaximumPayloadBytes == orderedtransportinstance.MaximumPayloadBytes && p.MaximumFrameBytes == orderedtransportinstance.MaximumFrameBytes && p.Receive.Identity != "" && p.Receive.Capability != "" && p.Receive.Sequence == 0 && p.Send.Identity != "" && p.Send.Capability != "" && p.Send.Sequence == 1 && p.Receive.Identity != p.Send.Identity && p.Receive.Capability != p.Send.Capability
}

func profileDigest(p Profile) [sha256.Size]byte {
	return sha256.Sum256([]byte(fmt.Sprintf("seme.ordered-transport.profile.v1\x00%q\x00%q\x00%d\x00%d\x00%q\x00%q\x00%d\x00%q\x00%q\x00%d", p.CodecIdentity, p.DigestIdentity, p.MaximumPayloadBytes, p.MaximumFrameBytes, p.Receive.Identity, p.Receive.Capability, p.Receive.Sequence, p.Send.Identity, p.Send.Capability, p.Send.Sequence)))
}
