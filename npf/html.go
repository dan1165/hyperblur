// Package npf renders Tumblr's Neue Post Format (NPF) into HTML.
//
// Go port of syeopite/npf-renderer (0.13.x line hyperblur pins), plus hyperblur
// tweaks. Output from tiny node tree, not ported `dominate` DOM. Keeps formatter
// close to original.
package npf

import (
	"sort"
	"strings"
)

// Node is minimal HTML element tree. Text nodes have empty Tag; Raw marks text
// unescaped on render.
type Node struct {
	Tag   string
	Attrs map[string]string
	Kids  []*Node
	Text  string
	Raw   bool
}

// El creates element. Attributes passed as alternating key/value pairs.
func El(tag string, attrs ...string) *Node {
	n := &Node{Tag: tag, Attrs: map[string]string{}}
	for i := 0; i+1 < len(attrs); i += 2 {
		n.SetAttr(attrs[i], attrs[i+1])
	}
	return n
}

// Txt creates escaped text node.
func Txt(s string) *Node { return &Node{Text: s} }

// RawHTML creates text node emitted verbatim.
func RawHTML(s string) *Node { return &Node{Text: s, Raw: true} }

// SetAttr sets attribute, normalising shorthand and dashes like dominate's
// clean_attribute.
func (n *Node) SetAttr(key, value string) *Node {
	if n.Attrs == nil {
		n.Attrs = map[string]string{}
	}
	n.Attrs[normalizeAttr(key)] = value
	return n
}

// Add appends children, returns receiver.
func (n *Node) Add(kids ...*Node) *Node {
	n.Kids = append(n.Kids, kids...)
	return n
}

// Prepend inserts child before all others.
func (n *Node) Prepend(kid *Node) *Node {
	n.Kids = append([]*Node{kid}, n.Kids...)
	return n
}

// findAll returns every descendant element with tag name, depth-first.
func (n *Node) findAll(tag string) []*Node {
	var out []*Node
	var rec func(*Node)
	rec = func(p *Node) {
		for _, c := range p.Kids {
			if c.Tag == "" {
				continue
			}
			if c.Tag == tag {
				out = append(out, c)
			}
			rec(c)
		}
	}
	rec(n)
	return out
}

// Find returns first descendant with tag, else nil.
func (n *Node) Find(tag string) *Node {
	if all := n.findAll(tag); len(all) > 0 {
		return all[0]
	}
	return nil
}

// FindClass returns first descendant with class attribute cls.
func (n *Node) FindClass(cls string) *Node {
	var found *Node
	var rec func(*Node)
	rec = func(p *Node) {
		if found != nil {
			return
		}
		for _, c := range p.Kids {
			if c.Tag == "" {
				continue
			}
			if c.Attrs["class"] == cls {
				found = c
				return
			}
			rec(c)
			if found != nil {
				return
			}
		}
	}
	rec(n)
	return found
}

// Render returns compact HTML for node.
func (n *Node) Render() string {
	var b strings.Builder
	n.render(&b)
	return b.String()
}

func (n *Node) render(b *strings.Builder) {
	if n == nil {
		return
	}
	if n.Tag == "" {
		if n.Raw {
			b.WriteString(n.Text)
		} else {
			b.WriteString(escape(n.Text))
		}
		return
	}

	b.WriteByte('<')
	b.WriteString(n.Tag)

	keys := make([]string, 0, len(n.Attrs))
	for k := range n.Attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteByte(' ')
		b.WriteString(k)
		b.WriteString(`="`)
		b.WriteString(escape(n.Attrs[k]))
		b.WriteByte('"')
	}
	b.WriteByte('>')

	if singleTags[n.Tag] {
		return
	}
	for _, c := range n.Kids {
		c.render(b)
	}
	b.WriteString("</")
	b.WriteString(n.Tag)
	b.WriteByte('>')
}

// escape escapes &, <, > and " (not '), like cgi.escape. Allocates nothing when
// nothing to escape.
func escape(s string) string {
	if !strings.ContainsAny(s, `&<>"`) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 8)
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

func normalizeAttr(key string) string {
	switch key {
	case "cls", "className", "class_name", "klass":
		return "class"
	case "fr", "html_for", "htmlFor":
		return "for"
	}
	if strings.HasPrefix(key, "data_") || strings.HasPrefix(key, "aria_") {
		return strings.ToLower(strings.ReplaceAll(key, "_", "-"))
	}
	if strings.HasPrefix(key, "_") {
		return key[1:]
	}
	return key
}

// singleTags render without closing tag, matching dominate's is_single.
var singleTags = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "command": true,
	"embed": true, "hr": true, "img": true, "input": true, "keygen": true,
	"link": true, "meta": true, "param": true, "source": true, "track": true,
	"wbr": true,
}
