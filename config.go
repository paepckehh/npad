package npad

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"os"
	"strings"
	"sync"

	"paepcke.de/logsec"
)

// global config
var c *Config

// internal
type intercom struct {
	// INTERNAL: INTERCOM
	store      map[string][]byte
	storeMUTEX sync.RWMutex
	// INTERNAL: KEYS AND STATES
	magicOFFSET    int
	plainOFFSET    int
	downloadOFFSET int
	qrOFFSET       int
	storeZERO      bool
	// INTRNAL: PRE COMPUTED WEB ELEMENTS
	head1  string
	head2  string
	head3  string
	head3b string
	home   string
	banner string
	url    string
}

func configure() {
	// init ram store backend [map]
	if !c.PermSTORE {
		c.i.store = make(map[string][]byte)
	}
	// pre-compute backend parameter
	if (c.Calgo == "GZIP" && c.Clevel > 9) || (c.Calgo == "ZSTD" && c.Clevel > 19) {
		panic("invalid compression level [" + c.Calgo + "] [" + itoa(c.Clevel) + "]")
	}
	c.i.downloadOFFSET = len(_download)
	c.i.plainOFFSET = len(_plain)
	c.i.magicOFFSET = len(_magic)
	c.i.qrOFFSET = len(_qr)
	e := c.Ealgo
	if c.Ealgo == "" {
		e = "DISABLED"
	}
	o := c.Calgo + ":" + itoa(c.Clevel)
	if c.Clevel == 0 {
		o = "DISABLED"
	}
	s := "NON-PERSISTENT:RAM-ONLY"
	ss := s
	if c.PermSTORE {
		s = "PERSISTENT:FS"
		ss = c.Chroot.DIR
	}
	n := "PLAINTEXT"
	proto := "http://"
	if c.CAcert != "" && c.CAkey != "" {
		n = "TLS13"
		proto = "https://"
	}
	if c.CAclient != "" {
		n += ":MTLS"
	}
	if c.CAPrivateOnly {
		n += ":ONLY"
	}
	c.i.url = proto + c.ListenAddr
	x := strings.Split(c.ListenAddr, ":")
	if len(x) == 2 {
		if x[1] == "80" && proto == "http://" {
			c.i.url = proto + x[0]
		}
		if x[1] == "443" && proto == "https://" {
			c.i.url = proto + x[0]
		}
	}
	// pre-compute ux components
	c.i.home = href + c.i.url + "\">" + "<button>" + home + " " + c.i.url + bue + "</a> "
	c.i.home += href + "/diag\">" + "<button>" + diag + bue + "</a> "
	c.i.home += href + "/src\">" + "<button>" + git + bue + "</a><br>"
	c.i.head1 = h1 + "\n\t<title>\n\t" + c.App + "\n\t</title>" + endHead
	c.i.head2 = h2 + "\n\t<title>\n\t" + c.App + "\n\t</title>" + endHead
	c.i.head3 = h2 + "\n\t<title>\n\t" + c.App + "\n\t</title>" + syntax_css + endHead
	c.i.head3b = h2b + "\n\t<title>\n\t" + c.App + "\n\t</title>" + syntax_css + endHead
	c.i.banner = c.i.home
	c.i.banner += bu + transport + " TRANSPORT:" + n + bue
	c.i.banner += bu + store + " STORE:" + s + ":" + o + bue
	c.i.banner += bu + lock + " ENCRYPT:" + e + bue + "<br>"
	// report configuration stats
	logsec.LogInfo <- "[" + c.ListenAddr + "] [TRANSPORT:" + n + "] [LOG:" + c.Log.LogMode + "]"
	logsec.LogInfo <- "[STORE:" + ss + "] [STORE:COMPRESS:" + o + "] [STORE:ENCRYPT:" + e + "]"
}

func getTLSConf() (listen net.Listener, err error) {
	if c.CAcert != "" && c.CAkey != "" {

		key, err := tls.LoadX509KeyPair(c.CAcert, c.CAkey)
		if err != nil {
			logsec.ShowErr("unable to [read|decode] [cert|key] [" + c.CAcert + "|" + c.CAkey + "] [" + err.Error() + "]")
			return listen, err
		}

		caClient := x509.NewCertPool()
		clientAuthMode := tls.VerifyClientCertIfGiven
		if c.CAclient != "" {
			cert, err := os.ReadFile(c.CAclient)
			if err != nil {
				logsec.ShowErr("unable to [read] clientCA cert [" + c.CAclient + "] [" + err.Error() + "]")
				return listen, err
			}
			caClient.AppendCertsFromPEM(cert)
			clientAuthMode = tls.VerifyClientCertIfGiven
			if c.CAPrivateOnly {
				clientAuthMode = tls.RequireAndVerifyClientCert
			}
		}

		tlsConf := &tls.Config{
			Certificates:           []tls.Certificate{key},
			ClientCAs:              caClient,
			ClientAuth:             clientAuthMode,
			MinVersion:             tls.VersionTLS13,
			MaxVersion:             tls.VersionTLS13,
			CipherSuites:           []uint16{tls.TLS_CHACHA20_POLY1305_SHA256},
			CurvePreferences:       []tls.CurveID{tls.X25519},
			NextProtos:             []string{"http/1.1"},
			SessionTicketsDisabled: true,
			Renegotiation:          0,
		}
		return tls.Listen("tcp", c.ListenAddr, tlsConf)
	}
	return net.Listen("tcp", c.ListenAddr)
}
