// package npad
package npad

// import
import (
	"time"

	"paepcke.de/logsec"
)

// Config global app configuration for examples values -> see APP folder
type Config struct {
	// ############################################### START APP CONFIG ###############################
	// APP
	App string // app name [required]
	// NETWORK
	ListenAddr string // server listen Address [name:port] [required]
	// TLS CERTIFICATES
	CAcert        string // server cert path [disable:<emptu>]
	CAkey         string // server key path [disable:<empty>]
	CAclient      string // clientCA certificate [disable: <empty>]
	CAPrivateOnly bool   // if true, enforce mtls mode-only [disable: false]
	// DATA STORE BACKEND
	Calgo     string        // compression algo  [GZIP] [extended:ZSTD, see io.go] ][disable <empty>]
	Clevel    int           // compression level [GZIP:1-9] [ZSTD:1-19] [disable: 0]
	Ealgo     string        // encryption algo [AESGCM|GCMSIV|X|CHACHA20POLY1305]
	AutoGC    bool          // enables removal of expired pastes [required]
	AutoGCInt time.Duration // config how often store gc is processed [required]
	// OPTIONAL PERMANENT DATA STORE FILE SYSTEM BACKEND
	// *** WARNING *** deactivated by default, if activated, stores pastes in <ChrootDir> instead of ram [map]!
	// *** WARNING *** any change or [de]activation of [encrypt|compress] parameter needs a complete permanent store wipe!
	PermSTORE bool // activeate the filesystem backed permanent store [disable: false]
	// Log
	Log *logsec.LogD // see pkg lib/logsec
	// Chroot
	Chroot *logsec.ChrootD // see pkg lib/logsec
	// ########################################## END APP CONFIG ######################################
	i intercom // internal process communication
}

// Start ...
func (conf *Config) Start() {
	c = conf
	start()
}
