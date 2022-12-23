// package npad ...
package npad

// import
import (
	"errors"
	"html"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"paepcke.de/certinfo"
	"paepcke.de/logsec"
	"paepcke.de/npad/syntax"
	"paepcke.de/npad/url2svg"
	"paepcke.de/npad/urlref"
	"paepcke.de/reportstyle"
	"paepcke.de/tlsinfo"
)

//
// HTML Page Renderer Engines
//

// var
var (
	certReportHTML = &certinfo.Report{
		Summary: true,
		OpenSSL: true,
		PEM:     false,
		Style:   styleHTML,
	}
	certReportText = &certinfo.Report{
		Summary: true,
		OpenSSL: true,
		PEM:     false,
		Style:   styleText,
	}
	styleHTML  = reportstyle.StyleHTML()
	styleText  = reportstyle.StyleText()
	errExpired = errors.New("paste expired")
)

// internalServerError ...
func internalServerError(r http.ResponseWriter) {
	http.Error(r, "Error: Internal Server Error (500)", http.StatusInternalServerError)
}

// getStartHTML ...
func getStartHTML() string {
	var s strings.Builder
	s.WriteString(c.i.head1)
	s.WriteString(body)
	s.WriteString(i1)
	s.WriteString(c.i.banner)
	s.WriteString(form)
	s.WriteString(endBody)
	return s.String()
}

// getPlainHTML ...
func getPlainHTML(key string) (string, error) {
	ts, isExpired := expired(key)
	if isExpired {
		return _empty, errExpired
	}
	p, err := readPaste(key, false)
	if err != nil {
		return _empty, err
	}
	var name string
	sp := strings.Split(key, "@")
	if len(sp) == 3 {
		name = sp[2] + _space
	}
	name += "[Size:" + hruIEC(uint64(len(p)), "byte") + "]\n\n"
	var s strings.Builder
	s.WriteString(c.i.head2)
	s.WriteString(body)
	s.WriteString(i2)
	s.WriteString(c.i.banner)
	s.WriteString(button(key, ts))
	s.WriteString(pre)
	s.WriteString(name)
	s.WriteString(html.EscapeString(p))
	s.WriteString(endPre)
	s.WriteString(endBody)
	return s.String(), nil
}

// getMagicHTML ...
func getMagicHTML(key string) (string, error) {
	ts, isExpired := expired(key)
	if isExpired {
		return _empty, errExpired
	}
	p, err := readPaste(key, false)
	if err != nil {
		return _empty, err
	}
	var name string
	sp := strings.Split(key, "@")
	if len(sp) == 3 {
		name = sp[2] + _space
	}
	name += "[Size:" + hruIEC(uint64(len(p)), "byte") + "]\n\n"
	var opt strings.Builder
	var certqr string
	switch len(p) > 32 && (strings.Contains(p[:32], "BEGIN ") || strings.Contains(p[:32], "ssh-")) {
	case true:
		if len(p) < 1024 {
			certqr = "<H2>Certificate QR</H2>" + url2svg.GetStringSVG(p)
		}
		opt.WriteString(pre)
		opt.WriteString(name)
		opt.WriteString(html.EscapeString(p))
		opt.WriteString("\n\t<H2>Certificate Status</H2><br>\n")
		opt.WriteString(endPre)
		opt.WriteString(certinfo.Decode(p, certReportHTML))
		opt.WriteString(pre)
		opt.WriteString(certqr)
		opt.WriteString(endPre)
	case false:
		opt.WriteString(preCSS)
		opt.WriteString(name)
		opt.WriteString(syntaxhl(p))
		opt.WriteString(urlref.GetURL(p, true, true))
		opt.WriteString(endPre)
	}
	var s strings.Builder
	s.WriteString(c.i.head3)
	s.WriteString(body)
	s.WriteString(i2)
	s.WriteString(c.i.banner)
	s.WriteString(button(key, ts))
	s.WriteString(opt.String())
	s.WriteString(endBody)
	return s.String(), nil
}

// getQRHTML ...
func getQRHTML(key string, urltarget *url.URL) (string, error) {
	ts, isExpired := expired(key)
	if isExpired {
		return _empty, errExpired
	}
	u := "<br><br><br><p style=\"font-size:0.5em\"></style><strong>"
	u += c.i.url + "/" + urltarget.String()[c.i.qrOFFSET:] + "</strong></p>"
	var s strings.Builder
	s.WriteString(c.i.head3b)
	s.WriteString(body)
	s.WriteString(i2)
	s.WriteString(c.i.banner)
	s.WriteString(button(key, ts))
	s.WriteString("<br><br><br>")
	s.WriteString(url2svg.GetSVG(urltarget))
	s.WriteString(u)
	s.WriteString(endBody)
	return s.String(), nil
}

// getDigagHTML provides the client connection analysis page
func getDiagHTML(q *http.Request) (string, error) {
	var opt strings.Builder
	opt.Grow(4 * 1024)
	opt.WriteString("\t<H2>client connection state</H2>\n")
	opt.WriteString(tlsinfo.ReportTlsState(q.TLS, styleHTML))
	opt.WriteString("\t<H2>complete raw request header</H2>\n")
	opt.WriteString(getDiagHTMLHeader(q))
	opt.WriteString("<H2><br><br><br>server timestamp [UTC] " + time.Now().Format(time.RFC3339) + "</H2>")
	var s strings.Builder
	s.Grow(8 * 1024)
	s.WriteString(c.i.head3)
	s.WriteString(body)
	s.WriteString(i3)
	s.WriteString(c.i.banner)
	s.WriteString(preCSS)
	s.WriteString(opt.String())
	s.WriteString(endPre)
	s.WriteString(endBody)
	return s.String(), nil
}

// getDiagHTMLHeader provides the sorted raw header summary
func getDiagHTMLHeader(q *http.Request) string {
	header := make([]string, len(q.Header))
	var i int
	for k, v := range q.Header {
		switch k {
		case "User-Agent", "Accept-Encoding":
			header[i] = li1 + pad(html.EscapeString(k)) + li0 + html.EscapeString(strings.Join(v, " ")) + lix
		default:
			header[i] = li1 + pad(html.EscapeString(k)) + li2 + html.EscapeString(strings.Join(v, " ")) + lix
		}
		i++
	}
	sort.Strings(header)
	var s strings.Builder
	for _, h := range header {
		s.WriteString(h)
	}
	return s.String()
}

//
// Plaintex ["curl"] Page Renderer Engine
//

// getPlainText ...
func getPlainText(key string) (string, error) {
	p, err := readPaste(key, false)
	if err != nil {
		return _empty, err
	}
	return p, nil
}

// getDiagText ...
func getDiagText(q *http.Request) (string, error) {
	var s strings.Builder
	s.Grow(4 * 1024)
	s.WriteString(tlsinfo.ReportTlsState(q.TLS, styleText) + _linefeed)
	s.WriteString(getDiagTextHeader(q) + _linefeed)
	return s.String(), nil
}

// getDiagTextHeader ...
func getDiagTextHeader(q *http.Request) string {
	var s strings.Builder
	s.Grow(2 * 1024)
	header := make([]string, len(q.Header))
	var i int
	for k, v := range q.Header {
		header[i] = pad(k) + " : " + strings.Join(v, _space)
		i++
	}
	sort.Strings(header)
	for _, h := range header {
		s.WriteString(h + _linefeed)
	}
	return s.String()
}

//
// Shared Functions
//

// syntaxhl wrapper
func syntaxhl(in string) string {
	p, err := syntax.AsHTML([]byte(in), syntax.OrderedList())
	if err != nil {
		logsec.LogErr <- err.Error() // blocks in case of global [err rate limit]
		return html.EscapeString(in)
	}
	return string(p)
}

// button shared button head
func button(key, ts string) string {
	var s strings.Builder
	s.WriteString(href + _plain[:c.i.plainOFFSET] + key + "\">" + bu + clip + " PLAIN TEXT" + bue + "</a>")
	s.WriteString(href + _magic[:c.i.magicOFFSET] + key + "\">" + bu + code + " MAGIC" + bue + "</a>")
	s.WriteString(bu + clock + " EXPIRE: " + ts + bue)
	s.WriteString(href + _download[:c.i.downloadOFFSET] + key + "\">" + bu + download + " DOWNLOAD" + bue + "</a>")
	s.WriteString(href + _qr[:c.i.qrOFFSET] + key + "\">" + bu + link + " QR" + bue + "</a>")
	return s.String()
}

// expired
func expired(key string) (string, bool) {
	ex := "NEVER"
	isExpired := false
	if key[:1] == "X" {
		if ts, _, ok := strings.Cut(key[1:], "@"); ok {
			x, err := atoi(ts)
			if err != nil {
				logsec.LogErr <- "expire time ascii -> int convert: [" + err.Error() + "]"
			}
			exp := time.Unix(int64(x), 0).Sub(time.Now())
			if exp < 1 {
				isExpired = true
			}
			ex = exp.Round(1 * time.Second).String()
			if exp > 259200000000000 {
				days := int(exp.Hours() / 24)
				ex = itoa(days) + " days"
			}
		}
	}
	return ex, isExpired
}

//
// Little Helper
//

// pad ...
func pad(in string) string {
	for len(in) < 26 {
		in = in + _space
	}
	return in
}

// bad ...
func bad(in bool) string {
	if in {
		return "YES [ALERT]"
	}
	return "NO [OK]"
}
