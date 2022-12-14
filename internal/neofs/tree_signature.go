/*REMOVE THIS AFTER SIGNATURE WILL BE AVAILABLE IN TREE CLIENT FROM NEOFS NODE*/
package neofs

import (
	crypto "github.com/TrueCloudLab/frostfs-crypto"
	"google.golang.org/protobuf/proto"
)

func (c *TreeClient) signData(buf []byte, f func(key, sign []byte)) error {
	// crypto package should not be used outside of API libraries (see neofs-node#491).
	// For now tree service does not include into SDK Client nor SDK Pool, so there is no choice.
	// When SDK library adopts Tree service client, this should be dropped.
	sign, err := crypto.Sign(&c.key.PrivateKey, buf)
	if err != nil {
		return err
	}

	f(c.key.PublicKey().Bytes(), sign)
	return nil
}

func (c *TreeClient) signRequest(requestBody proto.Message, f func(key, sign []byte)) error {
	buf, err := proto.Marshal(requestBody)
	if err != nil {
		return err
	}

	return c.signData(buf, f)
}
