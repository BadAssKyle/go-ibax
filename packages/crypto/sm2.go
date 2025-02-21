package crypto

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/tjfoc/gmsm/sm2"
	"github.com/IBAX-io/go-ibax/packages/consts"
	"github.com/IBAX-io/go-ibax/packages/converter"
)

type SM2 struct{}
	pubkeyCurve := sm2.P256Sm2()
	bi := new(big.Int).SetBytes(privateKey)
	priv := new(sm2.PrivateKey)
	priv.PublicKey.Curve = pubkeyCurve
	priv.D = bi
	priv.PublicKey.X, priv.PublicKey.Y = pubkeyCurve.ScalarBaseMult(bi.Bytes())
	ret, err := priv.Sign(rand.Reader, data, nil)
	return ret, err
}

func (s *SM2) verify(public, data, signature []byte) (bool, error) {
	if len(public) == 0 {
		return false, ErrCheckingSignEmpty
	}
	if len(data) == 0 {
		return false, fmt.Errorf("invalid parameters len(data) == 0")
	}
	if len(public) != consts.PubkeySizeLength {
		return false, fmt.Errorf("invalid parameters len(public) = %d", len(public))
	}
	if len(signature) == 0 {
		return false, fmt.Errorf("invalid parameters len(signature) == 0")
	}

	pubkeyCurve := sm2.P256Sm2()
	pubkey := new(sm2.PublicKey)
	pubkey.Curve = pubkeyCurve
	pubkey.X = new(big.Int).SetBytes(public[0:consts.PrivkeyLength])
	pubkey.Y = new(big.Int).SetBytes(public[consts.PrivkeyLength:])
	verifystatus := pubkey.Verify(data, signature)
	if !verifystatus {
		return false, ErrIncorrectSign
	}
	return true, nil
}

func (s *SM2) privateToPublic(key []byte) ([]byte, error) {
	pubkeyCurve := sm2.P256Sm2()
	bi := new(big.Int).SetBytes(key)
	priv := new(sm2.PrivateKey)
	priv.PublicKey.Curve = pubkeyCurve
	priv.D = bi
	priv.PublicKey.X, priv.PublicKey.Y = pubkeyCurve.ScalarBaseMult(key)
	return append(converter.FillLeft(priv.PublicKey.X.Bytes()), converter.FillLeft(priv.PublicKey.Y.Bytes())...), nil
}
