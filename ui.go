package npad

//
// compile time static [fast] UX stype elements
//

const (

	//
	// HTML
	//

	head     = "<html>\n<head>\n\t<style>\n\tbody{line-height:1.1;font-size:1.5em;text-align:center;background:#3367d6;color:#FFF}\n\tbutton{background:#3367d5;color:#FFF;border:none;border-radius:2px}\n\tbutton:hover{color:#3367d5;background:#FFF;}\n\t.i{stroke:currentColor;stroke-width:2px;stroke-linecap:round;stroke-linejoin:round;fill:none;width:1em;height:1em}\n\tsvg, svg symbol{overflow:visible;}"
	headb    = "<html>\n<head>\n\t<style>\n\tbody{line-height:1.1;font-size:1.5em;text-align:center;background:#000;color:#FFF}\n\tbutton{background:#3367d5;color:#FFF;border:none;border-radius:2px}\n\tbutton:hover{color:#3367d5;background:#FFF;}\n\t.i{stroke:currentColor;stroke-width:2px;stroke-linecap:round;stroke-linejoin:round;fill:none;width:1em;height:1em}\n\tsvg, svg symbol{overflow:visible;}"
	h1       = head + "\n\ttextarea{background:#3367d6;color:#FFF;}" + endStyle + icon
	h2       = head + "\n\tpre{font-size:0.6em;text-align:left}" + endStyle + icon
	h2b      = headb + "\n\tpre{font-size:0.6em;text-align:left;line-height:1.1;}" + endStyle + icon
	endHead  = "\n</head>\n"
	body     = "\n<body>\n"
	pre      = "\n<pre>\n"
	preCSS   = "\n<pre class=\"pr\" style=\"line-height:0.5;\">\n"
	endPre   = "\n</pre>\n"
	endStyle = "\n</style>\n"
	endBody  = "\n</body>\n</html>\n"
	form     = "<br>" + formdef + formbox + "\n\t</form>"
	formdef  = "\n\t<form action=\"/\" method=\"post\">"
	formbox  = input1 + "<br><br>" + input2 + "<br>" + input3 + "<br><br>" + expire + "<br>" + input4
	gorepo   = "paepcke.de/npad"
	input1   = "\n\t<textarea autofocus rows=\"46\" cols=\"80\" name=\"pa\"></textarea>"
	input2   = "<button style=\"padding:3px 2px\">[optional name]" + bue
	input3   = "<textarea rows=\"1\" cols=\"32\" name=\"na\" maxlength=\"32\"></textarea>"
	input4   = "\n\t<button type=\"submit\">" + up + bue
	expire   = "\n\t<SELECT NAME=\"ex\" style=\"font-size:0.7em;\" >" + exp0 + exp1 + exp2 + exp3 + "</style></SELECT><br>"
	exp0     = "<OPTION VALUE=\"0\">[Expire:20min][Max:10MB]"
	exp1     = "<OPTION VALUE=\"1\"SELECTED>[Expire:8hours][Max:8MB]"
	exp2     = "<OPTION VALUE=\"2\">[Expire:14days][Max:2MB]"
	exp3     = "<OPTION VALUE=\"3\">[Expire:Never][Max:200k]"
	bu       = "\n\t<button>"
	bue      = "</button>"
	href     = "\n\t<a href=\""
	disabled = "[function disabled or local database not ready yet]"
	icon     = "\n\t<link rel=\"icon\" type=\"image/svg+xml\" href=\"/i.svg\"/>"
	li1      = "\t<li><span class=\"kwd\">"
	li2      = "</span>:  <span class=\"str\">"
	li0      = "</span>:  <span class=\"fun\">"
	lix      = "</span></li>\n"
	space    = "\n\t<br><br><br>\n"

	//
	// CSS Stylesheets
	//

	// [css colors inspired by javascript parameter values github.com/jmblog/color-themes-for-google-code-prettify (MIT License)]
	syntax_css = "\n\t<style type = \"text/css\" media=\"screen\">.pr{background:#3367d5}.pln{color:#FFF}ol.linenums{color:#FFF}li.L0,li.L1,li.L2,li.L3,li.L4,li.L5,li.L6,li.L7,li.L8,li.L9{padding-left:1em;background-color:#3367d5;list-style-type:decimal}@media screen{.str{color:#b9ca4a}.kwd{color:#c397d8}.com{color:#969896}.typ{color:#7aa6da}.lit{color:#e78c45}.pun{color:#eaeaea}.opn{color:#eaeaea}.clo{color:#eaeaea}.tag{color:#d54e53}.atn{color:#e78c45}.atv{color:#70c0b1}.dec{color:#e78c45}.var{color:#d54e53}.fun{color:#7aa6da}}</style>"

	//
	// SVG [ux icon elements as embedded html code]
	//

	// svg icon sets
	i1 = _svgStart + _home + _ana + _git + _trans + _store + _lock + _up + _svgEnd
	i2 = _svgStart + _home + _ana + _git + _trans + _store + _lock + _link + _down + _clip + _code + _clock + _svgEnd
	i3 = _svgStart + _home + _ana + _git + _trans + _store + _lock + _svgEnd

	// shortcuts
	download  = _sc + "1" + _sx
	trash     = _sc + "2" + _sx
	code      = _sc + "3" + _sx
	home      = _sc + "4" + _sx
	transport = _sc + "5" + _sx
	store     = _sc + "6" + _sx
	clip      = _sc + "7" + _sx
	lock      = _sc + "8" + _sx
	link      = _sc + "9" + _sx
	git       = _sc + "X" + _sx
	diag      = _sc + "B" + _sx
	clock     = _sc + "D" + _sx
	up        = "<svg class=\"i\" style=\"font-size:10em\"><use xlink:href=\"#A" + _sx

	// svg icon code frags
	_svgStart = "\n\t<svg style=\"display:none\">"
	_svgEnd   = "</svg>"
	_sc       = "<svg class=\"i\"><use xlink:href=\"#"
	_sx       = "\"/></svg>"
	_s        = "<symbol id=\""
	_x        = "\" /></symbol>"
	_vb       = "\" viewBox=\"0 0 32 32\"><path d=\""
	_vx       = "\" viewBox=\"0 0 64 64\"><path "

	// svg design via https://github.com/danklammer/bytesize-icons [MIT]
	_down  = _s + "1" + _vb + "M9 22 C0 23 1 12 9 13 6 2 23 2 22 10 32 7 32 23 23 22 M11 26 L16 30 21 26 M16 16 L16 30" + _x
	_trash = _s + "2" + _vb + "M28 6 L6 6 8 30 24 30 26 6 4 6 M16 12 L16 24 M21 12 L20 24 M11 12 L12 24 M12 6 L13 2 19 2 20 6" + _x
	_code  = _s + "3" + _vb + "M10 9 L3 17 10 25 M22 9 L29 17 22 25 M18 7 L14 27" + _x
	_home  = _s + "4" + _vb + "M12 20 L12 30 4 30 4 12 16 2 28 12 28 30 20 30 20 20 Z" + _x
	_trans = _s + "5" + _vb + "M18 13 L26 2 8 13 14 19 6 30 24 19 Z" + _x
	_store = _s + "6" + _vb + "M4 10 L4 28 28 28 28 10 M2 4 L2 10 30 10 30 4 Z M12 15 L20 15" + _x
	_clip  = _s + "7" + _vb + "M12 2 L12 6 20 6 20 2 12 2 Z M11 4 L6 4 6 30 26 30 26 4 21 4" + _x
	_up    = _s + "A" + _vb + "M9 22 C0 23 1 12 9 13 6 2 23 2 22 10 32 7 32 23 23 22 M11 18 L16 14 21 18 M16 14 L16 29" + _x
	_ana   = _s + "B" + _vb + "M4 16 L11 16 14 29 18 3 21 16 28 16" + _x
	_clock = _s + "D" + "\" viewBox=\"0 0 32 32\"><circle cx=\"16\" cy=\"16\" r=\"14\" /><path d=\"M16 8 L16 16 20 20" + _x
	_lock  = _s + "8" + _vb + "M5 15 L5 30 27 30 27 15 Z M9 15 C9 9 9 5 16 5 23 5 23 9 23 15 M16 20 L16 23\" /><circle cx=\"16\" cy=\"24\" r=\"1" + _x
	_link  = _s + "9" + _vb + "M18 8 C18 8 24 2 27 5 30 8 29 12 24 16 19 20 16 21 14 17 M14 24 C14 24 8 30 5 27 2 24 3 20 8 16 13 12 16 11 18 15" + _x

	_git = _s + "X" + _vx + "stroke-width=\"0\" fill=\"currentColor\" d=\"M32 0 C14 0 0 14 0 32 0 53 19 62 22 62 24 62 24 61 24 60 L24 55 C17 57 14 53 13 50 13 50 13 49 11 47 10 46 6 44 10 44 13 44 15 48 15 48 18 52 22 51 24 50 24 48 26 46 26 46 18 45 12 42 12 31 12 27 13 24 15 22 15 22 13 18 15 13 15 13 20 13 24 17 27 15 37 15 40 17 44 13 49 13 49 13 51 20 49 22 49 22 51 24 52 27 52 31 52 42 45 45 38 46 39 47 40 49 40 52 L40 60 C40 61 40 62 42 62 45 62 64 53 64 32 64 14 50 0 32 0 Z" + _x

	_icon = "<svg xmlns=\"http://www.w3.org/2000/svg\" class=\"i\" viewBox=\"0 0 32 32\" width=\"64\" height=\"64\" fill=\"none\" stroke=\"currentcolor\" stroke-linecap=\"round\" stroke-linejoin=\"round\" stroke-width=\"2\"><path d=\"M27 15 L27 30 2 30 2 5 17 5 M30 6 L26 2 9 19 7 25 13 23 Z M22 6 L26 10 Z M9 19 L13 23 Z\"/></svg>"
)
