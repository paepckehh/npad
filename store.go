// package npad ...
package npad

// import
import (
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"paepcke.de/logsec"
	"paepcke.de/npad/compress"
	"paepcke.de/npad/encrypt"
)

//
// STORAGE BACKENDS
//

//
// Storage IO
//

// savePaste ...
func savePaste(in, name string, expire int) (string, error) {
	if name != "" {
		name = "@" + name
	}
	if len(name) > 32 {
		name = name[:32]
	}
	var prefix string
	switch expire {
	case 0:
		if len(in) > 10*1024*1024 {
			return "", errors.New("input to large - [allowed:20min|10MB]")
		}
		prefix = "X" + itoa64((time.Now().Add(20 * time.Minute)).Unix())
	case 1:
		if len(in) > 8*1024*1024 {
			return "", errors.New("input to large - [allowed:8h|8MB]")
		}
		prefix = "X" + itoa64((time.Now().Add(8 * time.Hour)).Unix())
	case 2:
		if len(in) > 2*1024*1024 {
			return "", errors.New("input to large - [allowed:14days|2MB]")
		}
		prefix = "X" + itoa64((time.Now().Add(14 * 24 * time.Hour)).Unix())
	case 3:
		if len(in) > 200*1024 {
			return "", errors.New("input to large - [allowed:256k]")
		}
		prefix = "N" + itoa64((time.Now()).Unix())
	default:
		panic("undefined store expire mode - unable to continue")
	}
	var err error
	var data []byte
	data = []byte(in)
	switch c.Clevel {
	case 0:
	default:
		data = compress.Compress(c.Calgo, c.Clevel, data)
	}
	k := sha512.Sum512(data)
	key := sha512.Sum384(append(append(k[:], strconv.FormatInt((time.Now().UnixNano()), 10)...), genRand()...))
	keyid := base64.RawURLEncoding.EncodeToString(key[:])
	file := prefix + "@" + keyid[:16] + name
	var url string
	switch c.Ealgo {
	case "":
		url = file
	default:
		rawkey := sha512.Sum512_256(key[16:])
		nonce := sha512.Sum512_224(append([]byte(prefix), key[:16]...))
		if data, err = encrypt.Encrypt(c.Ealgo, rawkey, nonce[:16], data); err != nil {
			return "", err
		}
		url = prefix + "@" + keyid + name
	}
	if c.PermSTORE {
		return url, os.WriteFile(file, data, 0o660)
	}
	c.i.storeMUTEX.Lock()
	c.i.store[file] = data
	c.i.storeMUTEX.Unlock()
	return url, nil
}

// readPaste ...
func readPaste(key string, raw bool) (string, error) {
	var (
		ok   bool
		err  error
		data []byte
	)
	var file string
	k := strings.Split(key, "@")
	switch len(k) {
	case 2:
		file = k[0] + "@" + k[1][:16]
	case 3:
		file = k[0] + "@" + k[1][:16] + "@" + k[2]
	default:
		return "", errors.New("[store] [url] [decode] [err]")
	}
	switch c.PermSTORE {
	case true:
		if data, err = os.ReadFile(file); err != nil {
			return "", err
		}
	case false:
		c.i.storeMUTEX.RLock()
		if data, ok = c.i.store[file]; !ok {
			c.i.storeMUTEX.RUnlock()
			return "", errors.New("key req miss [map] [" + file + "]")
		}
		c.i.storeMUTEX.RUnlock()
	}
	switch c.Ealgo {
	case "":
	default:
		key, err := base64.RawURLEncoding.DecodeString(k[1])
		if err != nil {
			return "", errors.New("decrypt url base64 decoder " + err.Error())
		}
		if len(key) != 48 {
			return "", errors.New("decrypt url key invalid")
		}
		rawkey := sha512.Sum512_256(key[16:])
		nonce := sha512.Sum512_224(append([]byte(k[0]), key[:16]...))
		if data, err = encrypt.Decrypt(c.Ealgo, rawkey, nonce[:16], data); err != nil {
			return "", err
		}
	}
	switch c.Clevel {
	case 0:
	default:
		if !raw {
			data = compress.Decompress(c.Calgo, data)
		}
	}
	return string(data), nil
}

//
// Storage GC
//

// isExpired
func isExpired(key string) bool {
	if key[:1] != "X" {
		return false
	}
	if ts, _, ok := strings.Cut(key[1:], "@"); ok {
		if t, err := atoi64(ts); err == nil {
			if t-time.Now().Unix() < 0 {
				return true
			}
		}
	}
	return false
}

// storeAutoGC
func storeAutoGC() {
	time.Sleep(12 * time.Second)
	for {
		switch c.PermSTORE {
		case true:
			gcFS()
		case false:
			gcMAP()
		default:
			panic("autogc permstore type")

		}
		runtime.GC()
		time.Sleep(20 * time.Minute)
	}
}

// gcMAAP ...
func gcMAP() {
	l := len(c.i.store)
	if l == 0 && c.i.storeZERO {
		return
	}
	if l == 0 {
		c.i.storeZERO = true
	}
	logsec.LogInfo <- "[gc] [ram] [pastes total:" + itoa(l) + "]"
	if l > 0 {
		c.i.storeZERO = false
		r := 0
		c.i.storeMUTEX.Lock()
		for k := range c.i.store {
			if isExpired(k) {
				delete(c.i.store, k)
				r++
				logsec.LogInfo <- "[gc] [removed] [" + k + "]"
			}
		}
		c.i.storeMUTEX.Unlock()
		logsec.LogInfo <- "[gc] [end] [ram-only] [pastes total:" + itoa(len(c.i.store)) + "] [removed:" + itoa(r) + "]"
	}
}

// gcFS ...
func gcFS() {
	c.i.storeZERO = false
	d, err := os.ReadDir(".")
	if err != nil {
		logsec.LogErr <- "[gc] [fs] [dir] " + err.Error()
		return
	}
	l := len(d)
	if l == 0 && c.i.storeZERO {
		return
	}
	if l == 0 {
		c.i.storeZERO = true
	}
	logsec.LogInfo <- ("[gc] [" + c.Chroot.DIR + "] [total:" + itoa(l) + "]")
	if l > 0 {
		c.i.storeZERO = false
		r := 0
		for _, key := range d {
			k := key.Name()
			if isExpired(k) {
				err := os.Remove(k)
				if err != nil {
					logsec.LogErr <- err.Error()
				} else {
					r++
					logsec.LogInfo <- "[gc] [removed] [" + k + "]"
				}
			}
		}
		d, err := os.ReadDir(".")
		if err != nil {
			logsec.LogErr <- "[gc] [fs] [dir] " + err.Error()
			return
		}
		logsec.LogInfo <- "[gc] [end] [" + c.Chroot.DIR + "] [total:" + itoa(len(d)) + "] [removed:" + itoa(r) + "]"
	}
}
