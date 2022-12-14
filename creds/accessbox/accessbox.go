package accessbox

import (
	"bytes"
	"crypto/cipher"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/TrueCloudLab/frostfs-sdk-go/bearer"
	"github.com/TrueCloudLab/frostfs-sdk-go/netmap"
	"github.com/TrueCloudLab/frostfs-sdk-go/session"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
	"google.golang.org/protobuf/proto"
)

// Box represents friendly AccessBox.
type Box struct {
	Gate     *GateData
	Policies []*ContainerPolicy
}

// ContainerPolicy represents friendly AccessBox_ContainerPolicy.
type ContainerPolicy struct {
	LocationConstraint string
	Policy             netmap.PlacementPolicy
}

// GateData represents gate tokens in AccessBox.
type GateData struct {
	AccessKey     string
	BearerToken   *bearer.Token
	SessionTokens []*session.Container
	GateKey       *keys.PublicKey
}

// NewGateData returns GateData from the provided bearer token and the public gate key.
func NewGateData(gateKey *keys.PublicKey, bearerTkn *bearer.Token) *GateData {
	return &GateData{GateKey: gateKey, BearerToken: bearerTkn}
}

// SessionTokenForPut returns the first suitable container session context for PUT operation.
func (g *GateData) SessionTokenForPut() *session.Container {
	return g.containerSessionToken(session.VerbContainerPut)
}

// SessionTokenForDelete returns the first suitable container session context for DELETE operation.
func (g *GateData) SessionTokenForDelete() *session.Container {
	return g.containerSessionToken(session.VerbContainerDelete)
}

// SessionTokenForSetEACL returns the first suitable container session context for SetEACL operation.
func (g *GateData) SessionTokenForSetEACL() *session.Container {
	return g.containerSessionToken(session.VerbContainerSetEACL)
}

func (g *GateData) containerSessionToken(verb session.ContainerVerb) *session.Container {
	for _, sessionToken := range g.SessionTokens {
		if isAppropriateContainerContext(sessionToken, verb) {
			return sessionToken
		}
	}
	return nil
}

func isAppropriateContainerContext(tok *session.Container, verb session.ContainerVerb) bool {
	switch verb {
	case session.VerbContainerSetEACL, session.VerbContainerDelete, session.VerbContainerPut:
		return tok.AssertVerb(verb)
	default:
		return false
	}
}

// Secrets represents AccessKey and the key to encrypt gate tokens.
type Secrets struct {
	AccessKey    string
	EphemeralKey *keys.PrivateKey
}

// Marshal returns the wire-format of AccessBox.
func (x *AccessBox) Marshal() ([]byte, error) {
	return proto.Marshal(x)
}

// Unmarshal parses the wire-format message and put data to x.
func (x *AccessBox) Unmarshal(data []byte) error {
	return proto.Unmarshal(data, x)
}

// PackTokens adds bearer and session tokens to BearerTokens and SessionToken lists respectively.
// Session token can be nil.
func PackTokens(gatesData []*GateData) (*AccessBox, *Secrets, error) {
	box := &AccessBox{}
	ephemeralKey, err := keys.NewPrivateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("create ephemeral key: %w", err)
	}
	box.OwnerPublicKey = ephemeralKey.PublicKey().Bytes()

	secret, err := generateSecret()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate accessKey as hex: %w", err)
	}

	if err := box.addTokens(gatesData, ephemeralKey, secret); err != nil {
		return nil, nil, fmt.Errorf("failed to add tokens to accessbox: %w", err)
	}

	return box, &Secrets{hex.EncodeToString(secret), ephemeralKey}, err
}

// GetTokens returns gate tokens from AccessBox.
func (x *AccessBox) GetTokens(owner *keys.PrivateKey) (*GateData, error) {
	sender, err := keys.NewPublicKeyFromBytes(x.OwnerPublicKey, elliptic.P256())
	if err != nil {
		return nil, fmt.Errorf("couldn't unmarshal OwnerPublicKey: %w", err)
	}
	ownerKey := owner.PublicKey().Bytes()
	for _, gate := range x.Gates {
		if !bytes.Equal(gate.GatePublicKey, ownerKey) {
			continue
		}

		gateData, err := decodeGate(gate, owner, sender)
		if err != nil {
			return nil, fmt.Errorf("failed to decode gate: %w", err)
		}
		return gateData, nil
	}

	return nil, fmt.Errorf("no gate data for key  %x was found", ownerKey)
}

// GetPlacementPolicy returns ContainerPolicy from AccessBox.
func (x *AccessBox) GetPlacementPolicy() ([]*ContainerPolicy, error) {
	var result []*ContainerPolicy
	for _, policy := range x.ContainerPolicy {
		var cnrPolicy ContainerPolicy
		if err := cnrPolicy.Policy.Unmarshal(policy.Policy); err != nil {
			return nil, fmt.Errorf("unmarshal placement policy: %w", err)
		}

		cnrPolicy.LocationConstraint = policy.LocationConstraint

		result = append(result, &cnrPolicy)
	}

	return result, nil
}

// GetBox parses AccessBox to Box.
func (x *AccessBox) GetBox(owner *keys.PrivateKey) (*Box, error) {
	tokens, err := x.GetTokens(owner)
	if err != nil {
		return nil, fmt.Errorf("get tokens: %w", err)
	}

	policy, err := x.GetPlacementPolicy()
	if err != nil {
		return nil, fmt.Errorf("get policy: %w", err)
	}

	return &Box{
		Gate:     tokens,
		Policies: policy,
	}, nil
}

func (x *AccessBox) addTokens(gatesData []*GateData, ephemeralKey *keys.PrivateKey, secret []byte) error {
	for _, gate := range gatesData {
		encBearer := gate.BearerToken.Marshal()
		encSessions := make([][]byte, len(gate.SessionTokens))
		for i, sessionToken := range gate.SessionTokens {
			encSessions[i] = sessionToken.Marshal()
		}

		tokens := new(Tokens)
		tokens.AccessKey = secret
		tokens.BearerToken = encBearer
		tokens.SessionTokens = encSessions

		boxGate, err := encodeGate(ephemeralKey, gate.GateKey, tokens)
		if err != nil {
			return fmt.Errorf("encode gate: %w", err)
		}
		x.Gates = append(x.Gates, boxGate)
	}
	return nil
}

func encodeGate(ephemeralKey *keys.PrivateKey, ownerKey *keys.PublicKey, tokens *Tokens) (*AccessBox_Gate, error) {
	data, err := proto.Marshal(tokens)
	if err != nil {
		return nil, fmt.Errorf("encode tokens: %w", err)
	}

	encrypted, err := encrypt(ephemeralKey, ownerKey, data)
	if err != nil {
		return nil, fmt.Errorf("ecrypt tokens: %w", err)
	}

	gate := new(AccessBox_Gate)
	gate.GatePublicKey = ownerKey.Bytes()
	gate.Tokens = encrypted
	return gate, nil
}

func decodeGate(gate *AccessBox_Gate, owner *keys.PrivateKey, sender *keys.PublicKey) (*GateData, error) {
	data, err := decrypt(owner, sender, gate.Tokens)
	if err != nil {
		return nil, fmt.Errorf("decrypt tokens: %w", err)
	}
	tokens := new(Tokens)
	if err = proto.Unmarshal(data, tokens); err != nil {
		return nil, fmt.Errorf("unmarshal tokens: %w", err)
	}

	var bearerTkn bearer.Token
	if err = bearerTkn.Unmarshal(tokens.BearerToken); err != nil {
		return nil, fmt.Errorf("unmarshal bearer token: %w", err)
	}

	sessionTkns := make([]*session.Container, len(tokens.SessionTokens))
	for i, encSessionToken := range tokens.SessionTokens {
		sessionTkn := new(session.Container)
		if err = sessionTkn.Unmarshal(encSessionToken); err != nil {
			return nil, fmt.Errorf("unmarshal session token: %w", err)
		}
		sessionTkns[i] = sessionTkn
	}

	gateData := NewGateData(owner.PublicKey(), &bearerTkn)
	gateData.SessionTokens = sessionTkns
	gateData.AccessKey = hex.EncodeToString(tokens.AccessKey)
	return gateData, nil
}

func generateShared256(prv *keys.PrivateKey, pub *keys.PublicKey) (sk []byte, err error) {
	if prv.PublicKey().Curve != pub.Curve {
		return nil, fmt.Errorf("not equal curves")
	}

	x, _ := pub.Curve.ScalarMult(pub.X, pub.Y, prv.D.Bytes())
	if x == nil {
		return nil, fmt.Errorf("shared key is point at infinity")
	}

	sk = make([]byte, 32)
	skBytes := x.Bytes()
	copy(sk[len(sk)-len(skBytes):], skBytes)
	return sk, nil
}

func deriveKey(secret []byte) ([]byte, error) {
	hash := sha256.New
	kdf := hkdf.New(hash, secret, nil, nil)
	key := make([]byte, 32)
	_, err := io.ReadFull(kdf, key)
	return key, err
}

func encrypt(owner *keys.PrivateKey, sender *keys.PublicKey, data []byte) ([]byte, error) {
	enc, err := getCipher(owner, sender)
	if err != nil {
		return nil, fmt.Errorf("get chiper: %w", err)
	}

	nonce := make([]byte, enc.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate random nonce: %w", err)
	}

	return enc.Seal(nonce, nonce, data, nil), nil
}

func decrypt(owner *keys.PrivateKey, sender *keys.PublicKey, data []byte) ([]byte, error) {
	dec, err := getCipher(owner, sender)
	if err != nil {
		return nil, fmt.Errorf("get chiper: %w", err)
	}

	if ld, ns := len(data), dec.NonceSize(); ld < ns {
		return nil, fmt.Errorf("wrong data size (%d), should be greater than %d", ld, ns)
	}

	nonce, cypher := data[:dec.NonceSize()], data[dec.NonceSize():]
	return dec.Open(nil, nonce, cypher, nil)
}

func getCipher(owner *keys.PrivateKey, sender *keys.PublicKey) (cipher.AEAD, error) {
	secret, err := generateShared256(owner, sender)
	if err != nil {
		return nil, fmt.Errorf("generate shared key: %w", err)
	}

	key, err := deriveKey(secret)
	if err != nil {
		return nil, fmt.Errorf("derive key: %w", err)
	}

	return chacha20poly1305.NewX(key)
}

func generateSecret() ([]byte, error) {
	b := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, b)
	return b, err
}
