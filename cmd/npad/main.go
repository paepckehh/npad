package main

import (
	"time"

	"paepcke.de/logsec"
	"paepcke.de/npad"
)

func main() {
	c := &npad.Config{
		App: "npad", // name [required]
		// NETWORK
		ListenAddr: "paste.paepcke.pnoc:443", // server listen Address [name:port] required]
		// TLS CERTIFICATES
		CAcert:        "/etc/app/npad/paste.pem", // server cert path [required]
		CAkey:         "/etc/app/npad/paste.key", // server key path [required]
		CAclient:      "/etc/ssl/clientCA.pem",   // clientCA certificate [disable: <empty>]
		CAPrivateOnly: false,                     // if true, enforce mtls mode-only [disable: false]
		// DATA STORE BACKEND any change or [de]activation of [compres|encrypt] parameter need a store wipe!
		Calgo:  "ZSTD", // compression algo  [GZIP|ZSTD] [disable <empty>]
		Clevel: 6,      // compression level [GZIP:1-9|ZSTD:1-19] [disable: 0]
		Ealgo:  "",     // encryption algo [AESGCM|GCMSIV|X|CHACHA20POLY1205] [disable: <empty>]
		// OPTIONAL PERMANENT DATA STORE FILE SYSTEM BACKEND
		// *** WARNING *** deactivated by default, if activated, stores pastes in <ChrootDir> instead of ram [map]!
		// *** WARNING *** any change or [de]activation of [encrypt|compress] parameter needs a complete permanent store wipe!
		PermSTORE: true, // activeate the filesystem backed permanent store [disable: false]
		// Log
		Log: &logsec.LogD{
			App:            "npad",                 // app log entries name
			LogMode:        "SYSLOG",               // write events to systemlog [SYSLOG|CONSOLE|FILE|MUTE]
			FileName:       "/var/log/npad.log",    // log file name
			ErrorRateLimit: 100 * time.Millisecond, // global error rate limit [ddos|error] protection timeout
		},
		// Chroot
		Chroot: &logsec.ChrootD{
			DIR: "/var/paste", // pastes ch-root & paste fs storeage directory [disable: <empty>]
			UID: 2004,         // chroot user UID number [disable: 0]
			GID: 2004,         // chroot user GID number [disable: 0]
		},
	}
	c.Start()
}
