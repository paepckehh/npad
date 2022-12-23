// package npad ...
package npad

// import
import (
	"crypto/rand"
	"fmt"
	"strconv"
)

//
// LITTLE HELPR
//

// genRand
func genRand() []byte {
	var rnd [64]byte
	_, _ = rand.Read(rnd[:])
	return rnd[:]
}

// itoa ...
func itoa(in int) string { return strconv.Itoa(in) }

// itoa64 ...
func itoa64(in int64) string { return strconv.FormatInt(in, 10) }

// itoaU64 ...
func itoaU64(in uint64) string { return strconv.FormatUint(in, 10) }

// atoi ..
func atoi(in string) (int, error) { return strconv.Atoi(in) }

// atoi64 ...
func atoi64(in string) (int64, error) { return strconv.ParseInt(in, 10, 0) }

// atoiU64 ...
func atoiU64(in string) (uint64, error) { return strconv.ParseUint(in, 10, 0) }

// HruIEC converts value to hru IEC 60027 units
func hruIEC(i uint64, u string) string {
	return hru(i, 1024, u)
}

// hru [human readable units] backend
func hru(i, unit uint64, u string) string {
	if i < unit {
		return fmt.Sprintf("%d %s", i, u)
	}
	div, exp := unit, 0
	for n := i / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	switch u {
	case "":
		return fmt.Sprintf("%.3f %c", float64(i)/float64(div), "kMGTPE"[exp])
	case "bit":
		return fmt.Sprintf("%.0f %c%s", float64(i)/float64(div), "kMGTPE"[exp], u)
	case "bytes", "bytes/sec":
		return fmt.Sprintf("%.1f %c%s", float64(i)/float64(div), "kMGTPE"[exp], u)
	}
	return fmt.Sprintf("%.3f %c%s", float64(i)/float64(div), "kMGTPE"[exp], u)
}
