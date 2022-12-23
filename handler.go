package npad

import (
	"net/http"
	"strings"

	"paepcke.de/logsec"
	"paepcke.de/npad/compress"
)

//
// MUX HANDLER FUNCTIONS
//

const (
	_ua         = "User-Agent"
	_utf8       = "text/html;charset=utf-8"
	_txt        = "text/plain"
	_svg        = "image/svg+xml"
	_ctype      = "Content-Type"
	_title      = "Title"
	_err_syntax = "[syntax] ["
	_err_plain  = "[plain] ["
)

func _headPlain(r http.ResponseWriter) http.ResponseWriter {
	r.Header().Set(_ctype, _txt)
	r.Header().Set(_title, c.App)
	return r
}

func _headHTML(r http.ResponseWriter) http.ResponseWriter {
	r.Header().Set(_ctype, _utf8)
	r.Header().Set(_title, c.App)
	return r
}

func _headSVG(r http.ResponseWriter) http.ResponseWriter {
	r.Header().Set(_ctype, _svg)
	r.Header().Set(_title, c.App)
	return r
}

// plain text display
func getPlainHandler() http.Handler {
	h := func(r http.ResponseWriter, q *http.Request) {
		var err error
		page := ""
		switch {
		case strings.Contains(strings.Join(q.Header[_ua], " "), "curl"):
			r = _headPlain(r)
			page, err = getPlainText(q.URL.Path[c.i.plainOFFSET:])
		default:
			r = _headHTML(r)
			page, err = getPlainHTML(q.URL.Path[c.i.plainOFFSET:])
		}
		if err != nil {
			logsec.LogErr <- _err_plain + err.Error() + "]"
			http.NotFound(r, q)
			return
		}
		compress.WriteTransportCompressedPage(page, r, q, true)
	}
	return http.HandlerFunc(h)
}

// syntax display
func getMagicHandler() http.Handler {
	h := func(r http.ResponseWriter, q *http.Request) {
		r = _headHTML(r)
		page, err := getMagicHTML(q.URL.Path[c.i.magicOFFSET:])
		if err != nil {
			logsec.LogErr <- _err_syntax + err.Error() + "]"
			http.NotFound(r, q)
			return
		}
		compress.WriteTransportCompressedPage(page, r, q, true)
	}
	return http.HandlerFunc(h)
}

// qr code
func getQRHandler() http.Handler {
	h := func(r http.ResponseWriter, q *http.Request) {
		r = _headHTML(r)
		page, err := getQRHTML(q.URL.Path[c.i.qrOFFSET:], q.URL)
		if err != nil {
			logsec.LogErr <- "[qr] [" + err.Error() + "]"
			http.NotFound(r, q)
			return
		}
		compress.WriteTransportCompressedPage(page, r, q, true)
	}
	return http.HandlerFunc(h)
}

// raw download
func getDownloadHandler() http.Handler {
	h := func(r http.ResponseWriter, q *http.Request) {
		page, err := readPaste(q.URL.Path[c.i.downloadOFFSET:], true)
		if err != nil {
			logsec.LogErr <- err.Error()
			http.NotFound(r, q)
			return
		}
		name := q.URL.Path[c.i.downloadOFFSET+1:]
		s := strings.Split(name, "@")
		if len(s) == 3 {
			name = s[2]
		}
		r.Header().Set("Content-disposition", "attachment; filename="+name+compress.GetFileExtension(c.Calgo))
		compress.WriteTransportCompressedPage(page, r, q, false)
	}
	return http.HandlerFunc(h)
}

// input ["start"] page
func getStartHandler() http.Handler {
	h := func(r http.ResponseWriter, q *http.Request) {
		r = _headHTML(r)
		switch q.Method {
		case "GET":
			compress.WriteTransportCompressedPage(getStartHTML(), r, q, true)
		case "POST":
			err := q.ParseForm()
			if err != nil {
				logsec.LogErr <- err.Error() // blocks in case of global [ddos|err rate limit]
				internalServerError(r)
				return
			}
			expire, err := atoi(q.FormValue("ex"))
			if err != nil {
				logsec.LogInfo <- "[new] [parse form data] [invalid expire option] " + err.Error()
				internalServerError(r)
				return
			}
			newKey, err := savePaste(q.FormValue("pa"), q.FormValue("na"), expire)
			if err != nil {
				logsec.LogInfo <- "[new] [save paste] " + err.Error()
				internalServerError(r)
				return
			}
			logsec.LogInfo <- "[new] " + newKey // optional log info event
			http.Redirect(r, q, _plain+newKey, http.StatusFound)
		default:
			inf := "Error: Method Not Allowed (405) [" + q.Method + "]"
			logsec.LogErr <- "[handler] [/] [" + inf + "]"
			http.Error(r, inf, http.StatusMethodNotAllowed)
		}
	}
	return http.HandlerFunc(h)
}

// client connection diagnosis
func getDiagHandler() http.Handler {
	h := func(r http.ResponseWriter, q *http.Request) {
		var err error
		page := ""
		switch {
		case strings.Contains(strings.Join(q.Header[_ua], " "), "curl"):
			r = _headPlain(r)
			page, err = getDiagText(q)
		default:
			_headHTML(r)
			page, err = getDiagHTML(q)
		}
		if err != nil {
			logsec.LogErr <- "[handler] [/diag] [" + err.Error() + "]"
		}
		compress.WriteTransportCompressedPage(page, r, q, true)
	}
	return http.HandlerFunc(h)
}

// app source code forwarder
func getSourceCodeHandler() http.Handler {
	h := func(r http.ResponseWriter, q *http.Request) {
		http.Redirect(r, q, "https://"+gorepo, http.StatusTemporaryRedirect)
	}
	return http.HandlerFunc(h)
}

// favicon [shared target cross all pages]
func getFavIconHandler() http.Handler {
	h := func(r http.ResponseWriter, q *http.Request) {
		r = _headSVG(r)
		compress.WriteTransportCompressedPage(_icon, r, q, true)
	}
	return http.HandlerFunc(h)
}
