package accessbox

import (
	"testing"

	"github.com/TrueCloudLab/frostfs-sdk-go/bearer"
	neofsecdsa "github.com/TrueCloudLab/frostfs-sdk-go/crypto/ecdsa"
	"github.com/TrueCloudLab/frostfs-sdk-go/eacl"
	"github.com/TrueCloudLab/frostfs-sdk-go/session"
	"github.com/google/uuid"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/stretchr/testify/require"
)

func assertBearerToken(t *testing.T, exp, act bearer.Token) {
	// compare binary representations since deep equal is not guaranteed
	require.Equal(t, exp.Marshal(), act.Marshal())
}

func TestTokensEncryptDecrypt(t *testing.T) {
	var (
		tkn  bearer.Token
		tkn2 bearer.Token
	)
	sec, err := keys.NewPrivateKey()
	require.NoError(t, err)

	cred, err := keys.NewPrivateKey()
	require.NoError(t, err)

	tkn.SetEACLTable(*eacl.NewTable())
	require.NoError(t, tkn.Sign(sec.PrivateKey))

	data, err := encrypt(cred, cred.PublicKey(), tkn.Marshal())
	require.NoError(t, err)

	rawTkn2, err := decrypt(cred, cred.PublicKey(), data)
	require.NoError(t, err)

	err = tkn2.Unmarshal(rawTkn2)
	require.NoError(t, err)

	assertBearerToken(t, tkn, tkn2)
}

func TestBearerTokenInAccessBox(t *testing.T) {
	var (
		box  *AccessBox
		box2 AccessBox
		tkn  bearer.Token
	)

	sec, err := keys.NewPrivateKey()
	require.NoError(t, err)

	cred, err := keys.NewPrivateKey()
	require.NoError(t, err)

	tkn.SetEACLTable(*eacl.NewTable())
	require.NoError(t, tkn.Sign(sec.PrivateKey))

	gate := NewGateData(cred.PublicKey(), &tkn)
	box, _, err = PackTokens([]*GateData{gate})
	require.NoError(t, err)

	data, err := box.Marshal()
	require.NoError(t, err)

	err = box2.Unmarshal(data)
	require.NoError(t, err)

	tkns, err := box2.GetTokens(cred)
	require.NoError(t, err)

	assertBearerToken(t, tkn, *tkns.BearerToken)
}

func TestSessionTokenInAccessBox(t *testing.T) {
	var (
		box  *AccessBox
		box2 AccessBox
		tkn  = new(session.Container)
	)

	sec, err := keys.NewPrivateKey()
	require.NoError(t, err)

	cred, err := keys.NewPrivateKey()
	require.NoError(t, err)

	tkn.SetID(uuid.New())
	tkn.SetAuthKey((*neofsecdsa.PublicKey)(sec.PublicKey()))
	require.NoError(t, tkn.Sign(sec.PrivateKey))

	var newTkn bearer.Token
	gate := NewGateData(cred.PublicKey(), &newTkn)
	gate.SessionTokens = []*session.Container{tkn}
	box, _, err = PackTokens([]*GateData{gate})
	require.NoError(t, err)

	data, err := box.Marshal()
	require.NoError(t, err)

	err = box2.Unmarshal(data)
	require.NoError(t, err)

	tkns, err := box2.GetTokens(cred)
	require.NoError(t, err)

	require.Equal(t, []*session.Container{tkn}, tkns.SessionTokens)
}

func TestAccessboxMultipleKeys(t *testing.T) {
	var (
		box *AccessBox
		tkn bearer.Token
	)

	sec, err := keys.NewPrivateKey()
	require.NoError(t, err)

	tkn.SetEACLTable(*eacl.NewTable())
	require.NoError(t, tkn.Sign(sec.PrivateKey))

	count := 10
	gates := make([]*GateData, 0, count)
	privateKeys := make([]*keys.PrivateKey, 0, count)
	{ // generate keys
		for i := 0; i < count; i++ {
			cred, err := keys.NewPrivateKey()
			require.NoError(t, err)

			gates = append(gates, NewGateData(cred.PublicKey(), &tkn))
			privateKeys = append(privateKeys, cred)
		}
	}

	box, _, err = PackTokens(gates)
	require.NoError(t, err)

	for i, k := range privateKeys {
		tkns, err := box.GetTokens(k)
		require.NoError(t, err, "key #%d: %s failed", i, k)
		assertBearerToken(t, tkn, *tkns.BearerToken)
	}
}

func TestUnknownKey(t *testing.T) {
	var (
		box *AccessBox
		tkn bearer.Token
	)

	sec, err := keys.NewPrivateKey()
	require.NoError(t, err)

	cred, err := keys.NewPrivateKey()
	require.NoError(t, err)

	wrongCred, err := keys.NewPrivateKey()
	require.NoError(t, err)

	tkn.SetEACLTable(*eacl.NewTable())
	require.NoError(t, tkn.Sign(sec.PrivateKey))

	gate := NewGateData(cred.PublicKey(), &tkn)
	box, _, err = PackTokens([]*GateData{gate})
	require.NoError(t, err)

	_, err = box.GetTokens(wrongCred)
	require.Error(t, err)
}
