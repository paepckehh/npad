// [experimental] [in-progress] [benchmark|power-usage] - do not use yet anywhere
package npad

import "time"

//
// Storage Feature BitMask
//

/* persistent storage bitmask
 * 00000000000
 * ||||||||||| EXPIRE
 * ||||||||||  AESGCM
 * |||||||||   GCMSIV
 * ||||||||    CHACHA20POLY1305
 * |||||||     XCHACHA20POLY1305
 * ||||||      GZIP
 * |||||       ZSTD
 * ||||        ZSTDDICT
 * |||         LZ4
 * ||          XZ
 */

const (
	EXPIRE = 1 << iota
	AESGCM
	GCMSIV
	CHACHA20POLY1305
	XCHACHA20POLY1305
	GZIP
	ZSTD
	ZSTDDICT
	LZ4
	XZ
)

func getCurrentStorageMask(c *Config) (bitmask uint) {
	// EXPIRE
	switch c.PasteLifeTime {
	case time.Duration(0 * time.Second):
	default:
		bitmask += EXPIRE
	}
	// ENCRYPT
	switch c.ekeyHASH {
	case [32]byte{}: // none
	default:
		switch c.Ealgo {
		case "AESGCM":
			bitmask += AESGCM
		case "GCMSIV":
			bitmask += GCMSIV
		case "CHACHA20POLY1305":
			bitmask += CHACHA20POLY1305
		case "XCHACHA20POLY1305":
			bitmask += XCHACHA20POLY1305
		default:
			panic("unsupported encryption algo [" + c.Ealgo + "]")
		}
	}
	// COMPRESS
	switch c.Clevel {
	case 0: // none
	default:
		switch c.Calgo {
		case "GZIP":
			bitmask += GZIP
		case "ZSTD":
			bitmask += ZSTD
		case "ZSTDDICT":
			bitmask += ZSTDDICT
		case "LZ4":
			bitmask += LZ4
		case "XZ":
			bitmask += XZ
		default:
			panic("unsupported compression algo [" + c.Calgo + "]")
		}
	}
	return bitmask
}

func resolveStorageMask(bitmask uint) (expire bool, eAlgo, cAlgo string) {
	expire = false
	switch {
	case (bitmask & EXPIRE) != 0:
		expire = true
	case (bitmask & AESGCM) != 0:
		eAlgo = "AESGCM"
	case (bitmask & GCMSIV) != 0:
		eAlgo = "GCMSIV"
	case (bitmask & CHACHA20POLY1305) != 0:
		eAlgo = "CHACHA20POLY1305)"
	case (bitmask & XCHACHA20POLY1305) != 0:
		eAlgo = "XCHACHA20POLY1305)"
	case (bitmask & GZIP) != 0:
		cAlgo = "GZIP"
	case (bitmask & ZSTD) != 0:
		cAlgo = "ZSTD"
	case (bitmask & ZSTDDICT) != 0:
		cAlgo = "ZSTDDICT"
	case (bitmask & LZ4) != 0:
		cAlgo = "LZ4"
	case (bitmask & XZ) != 0:
		cAlgo = "XZ"
	}
	return
}
