package markup

import "strings"

// FromDText converts a booru's DText body into the vocabulary Parse reads.
// Danbooru and e621 publish artist commentary as DText rather than HTML, so
// this is the commentary's FromHTML: the inline marks the two spellings
// already share pass through untouched, the wrappers this side has no mark for
// keep their text and lose their brackets, and DText's link and wiki forms
// become the link constructs. Anything else stays the characters it is, the
// way Parse itself degrades.
func FromDText(src string) string {
	d := &dtext{}
	d.run(src)
	return strings.TrimSpace(d.out.String())
}

type dtext struct {
	out strings.Builder
	// inLink suppresses a second link inside one, which Parse refuses; the
	// label survives on its own.
	inLink bool
	depth  int
}

// dtextWrappers are the block constructs with no mark on this side. Their text
// is what the reader wants and the brackets are furniture.
var dtextWrappers = map[string]bool{
	"quote": true, "expand": true, "section": true, "spoiler": true,
	"color": true, "nodtext": true,
	"table": true, "thead": true, "tbody": true, "tr": true, "th": true, "td": true,
}

func (d *dtext) run(src string) {
	for i := 0; i < len(src); {
		if d.atLineStart() {
			if w := headerWidth(src[i:]); w > 0 {
				i += w
				continue
			}
		}
		if w := d.construct(src[i:]); w > 0 {
			i += w
			continue
		}
		if src[i] == '\r' {
			d.out.WriteByte('\n')
			if i+1 < len(src) && src[i+1] == '\n' {
				i++
			}
			i++
			continue
		}
		d.out.WriteByte(src[i])
		i++
	}
}

// construct reads one DText form at the head of s. A zero width means s opens
// none and its first byte is ordinary text.
func (d *dtext) construct(s string) int {
	switch {
	case strings.HasPrefix(s, "[["):
		return d.wikiLink(s)
	case s[0] == '[':
		return d.bracketTag(s)
	case strings.HasPrefix(s, "{{"):
		return d.tagSearch(s)
	case s[0] == '"':
		return d.quotedLink(s)
	case s[0] == '<':
		return d.angleURL(s)
	}
	return d.bareURL(s)
}

// bracketTag handles the [name] forms: a mark both vocabularies spell the same
// way survives verbatim, [br] is a newline, a wrapper loses its brackets, and
// anything else is left as the characters it is.
func (d *dtext) bracketTag(s string) int {
	end := strings.IndexByte(s, ']')
	if end < 0 {
		return 0
	}
	inner := s[1:end]
	closing := strings.HasPrefix(inner, "/")
	name := strings.TrimPrefix(inner, "/")
	// [expand=title] and [section,expanded=true] name the construct first.
	if cut := strings.IndexAny(name, "=,"); cut >= 0 {
		name = name[:cut]
	}
	name = strings.ToLower(strings.TrimSpace(name))
	switch {
	case marks.has(name) && (inner == name || inner == "/"+name):
		d.out.WriteString(s[:end+1])
	case name == "br" && !closing:
		d.out.WriteString("\n")
	case dtextWrappers[name]:
	default:
		return 0
	}
	return end + 1
}

// wikiLink converts [[page]] and [[page|label]]. A danbooru wiki page is a
// tag, written with the spaces a tag name carries as underscores.
func (d *dtext) wikiLink(s string) int {
	end := strings.Index(s, "]]")
	if end < 0 {
		return 0
	}
	page, label := splitRef(s[2:end])
	d.tagRef(strings.ReplaceAll(strings.TrimSpace(page), " ", "_"), label)
	return end + 2
}

// tagSearch converts {{tag}}. A search of several terms names no one tag, so
// it keeps its text.
func (d *dtext) tagSearch(s string) int {
	end := strings.Index(s, "}}")
	if end < 0 {
		return 0
	}
	terms, label := splitRef(s[2:end])
	d.tagRef(strings.TrimSpace(terms), label)
	return end + 2
}

// quotedLink converts "label":url and "label":[url]. The colon has to be
// followed by something link-shaped or an ordinary quoted phrase before one
// would become a link.
func (d *dtext) quotedLink(s string) int {
	q := strings.IndexByte(s[1:], '"')
	if q < 0 {
		return 0
	}
	label := s[1 : 1+q]
	if strings.ContainsAny(label, "\r\n") {
		return 0
	}
	rest := s[q+2:]
	if !strings.HasPrefix(rest, ":") {
		return 0
	}
	rest = rest[1:]
	var href string
	var w int
	if strings.HasPrefix(rest, "[") {
		end := strings.IndexByte(rest, ']')
		if end < 0 {
			return 0
		}
		href, w = rest[1:end], end+1
	} else if w = urlRunLen(rest); w > 0 {
		href = rest[:w]
	} else {
		return 0
	}
	d.urlRef(href, label)
	return q + 3 + w
}

// angleURL converts <https://...>, which is how DText links a URL carrying
// punctuation a bare one would stop at.
func (d *dtext) angleURL(s string) int {
	end := strings.IndexByte(s, '>')
	if end < 0 || !validURL(strings.TrimSpace(s[1:end])) {
		return 0
	}
	d.urlRef(s[1:end], s[1:end])
	return end + 1
}

// bareURL links a URL written on its own, which is how most commentary carries
// one. It has to start a word, or a URL glued to the end of one would link.
func (d *dtext) bareURL(s string) int {
	if !validURL(s) || !d.atWordStart() {
		return 0
	}
	n := urlRunLen(s)
	if n == 0 {
		return 0
	}
	d.urlRef(s[:n], s[:n])
	return n
}

// tagRef emits a tag reference, or the label alone when the name is not one
// tag this side could resolve.
func (d *dtext) tagRef(name, label string) {
	if d.inLink || !isTagRef(name) {
		d.label(label)
		return
	}
	d.out.WriteString("[tag=" + name + "]")
	d.inLink = true
	d.label(label)
	d.inLink = false
	d.out.WriteString("[/tag]")
}

// urlRef emits a link, or the label alone for a target Parse could not render
// as one - a site-relative href with no host to resolve it against, above all.
func (d *dtext) urlRef(href, label string) {
	href = strings.TrimSpace(href)
	if d.inLink || !validURL(href) || strings.ContainsAny(href, " \t\r\n[]") {
		d.label(label)
		return
	}
	d.out.WriteString("[url=" + href + "]")
	d.inLink = true
	d.label(label)
	d.inLink = false
	d.out.WriteString("[/url]")
}

// label converts a reference's visible text, which carries marks of its own.
// Past the nesting cap it is written as it stands, like any other run the
// converter does not take apart.
func (d *dtext) label(s string) {
	if d.depth >= maxDepth {
		d.out.WriteString(s)
		return
	}
	d.depth++
	d.run(s)
	d.depth--
}

func (d *dtext) atLineStart() bool {
	s := d.out.String()
	return s == "" || strings.HasSuffix(s, "\n")
}

func (d *dtext) atWordStart() bool {
	s := d.out.String()
	if s == "" {
		return true
	}
	b := s[len(s)-1]
	return isSpace(b) || strings.IndexByte("([{<\"'", b) >= 0
}

// splitRef cuts a wiki or search reference into its target and the label it is
// written with, which is the target itself when none is given.
func splitRef(s string) (target, label string) {
	if i := strings.IndexByte(s, '|'); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, s
}

// urlRunLen measures the URL at the head of s. It ends at whitespace or a
// character DText writes around a link, and the sentence punctuation a URL is
// followed by is not part of it.
func urlRunLen(s string) int {
	i := 0
	for i < len(s) && !isSpace(s[i]) && strings.IndexByte(`<>"[]`, s[i]) < 0 {
		i++
	}
	for i > 0 && strings.IndexByte(".,;:!?", s[i-1]) >= 0 {
		i--
	}
	return i
}

// headerWidth measures an hN. header prefix, which carries no text of its own.
func headerWidth(s string) int {
	if len(s) < 3 || s[0] != 'h' || s[1] < '1' || s[1] > '6' {
		return 0
	}
	i := 2
	// h4#anchor. names a link target the body has no use for.
	if s[i] == '#' {
		for i < len(s) && s[i] != '.' && !isSpace(s[i]) {
			i++
		}
	}
	if i >= len(s) || s[i] != '.' {
		return 0
	}
	for i++; i < len(s) && s[i] == ' '; i++ {
	}
	return i
}
