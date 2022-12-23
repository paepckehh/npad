package npad

import (
	"net/http"

	"paepcke.de/logsec"
)

const (
	_root     = "/"
	_qr       = "/qr/"
	_plain    = "/plain/"
	_magic    = "/magic/"
	_diag     = "/diag/"
	_src      = "/src/"
	_favicon  = "/i.svg"
	_download = "/download/"
	_empty    = ""
	_linefeed = "\n"
	_space    = " "
)

// start ...
func start() {
	// start logging
	logsec.LogDaemon(c.Log)

	// bind ports before priv drop
	listener, err := getTLSConf()
	if err != nil {
		logsec.ShowErr("[fatal] unable to bind to address [" + c.ListenAddr + "] [" + err.Error() + "]")
		return
	}

	// drop privs
	if logsec.Chroot(c.Chroot) {

		// setup mux
		mux := http.NewServeMux()

		// handler
		mux.Handle(_root, getStartHandler())
		mux.Handle(_download, getDownloadHandler())
		mux.Handle(_qr, getQRHandler())
		mux.Handle(_plain, getPlainHandler())
		mux.Handle(_magic, getMagicHandler())
		mux.Handle(_diag, getDiagHandler())
		mux.Handle(_src, getSourceCodeHandler())
		mux.Handle(_favicon, getFavIconHandler())

		//
		httpsrv := &http.Server{
			Handler: mux,
		}

		// store gc
		go storeAutoGC()

		// setup keys, store, ux elements
		configure()

		// serve requestes
		err = httpsrv.Serve(listener)
		e := ("[shutdown] [fatal] [server error] [" + err.Error() + "]") // no recover after priv drop & crash
		logsec.LogErr <- e
		logsec.ShowErr(e)
	}
}
