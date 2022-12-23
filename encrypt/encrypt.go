// package encrypt  ...
package encrypt

// import
import (
	"crypto/aes"
	"crypto/cipher"
	"errors"

	"paepcke.de/npad/encrypt/gcmsiv"
	// "github.com/paepckehh/contrib_agl_gcmsiv/v1" 
	// "github.com/ag/gcmsiv" needs upstream fix for 32 bit (embedded) os plattofrm support 
)

// Encrypt ...
func Encrypt(algo string, key [32]byte, iv, data []byte) ([]byte, error) {
	if key == [32]byte{} || len(iv) < 16 {
		return []byte{}, errors.New("[enc] [active:" + algo + "] [but no valid keys]")
	}
	var err error
	var x cipher.AEAD
	switch algo {
	case "GCMSIV":
		x, err = gcmsiv.NewGCMSIV(key[:])
	case "AESGCM":
		e, err := aes.NewCipher(key[:])
		if err != nil {
			return []byte{}, err
		}
		x, err = cipher.NewGCM(e)
		if err != nil {
			return []byte{}, err
		}
	default:
		panic("unsupported encryption algo [" + algo + "]")
	}
	if err != nil {
		return []byte{}, err
	}
	return x.Seal(nil, iv[:x.NonceSize()], data, nil), nil
}

// Decrypt ...
func Decrypt(algo string, key [32]byte, iv, data []byte) ([]byte, error) {
	if key == [32]byte{} || len(iv) < 16 {
		return []byte{}, errors.New("[enc] [active:" + algo + "] [but no valid keys]")
	}
	var err error
	var x cipher.AEAD
	switch algo {
	case "GCMSIV":
		x, err = gcmsiv.NewGCMSIV(key[:])
	case "AESGCM":
		d, err := aes.NewCipher(key[:])
		if err != nil {
			return []byte{}, err
		}
		x, err = cipher.NewGCM(d)
		if err != nil {
			return []byte{}, err
		}
	default:
		panic("unsupported encryption algo [" + algo + "]")
	}
	if err != nil {
		return []byte{}, err
	}
	return x.Open(nil, iv[:x.NonceSize()], data, nil)
}
