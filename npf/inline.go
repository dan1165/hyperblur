package npf

import "sort"

// parseInlineText converts raw NPF inline formatting intervals into a sorted,
// non-overlapping set of style intervals. This replaces npf-renderer's use of
// an interval tree with an equivalent boundary split.
func (p *Parser) parseInlineText(raw []any) []StyleInterval {
	type iv struct {
		begin, end int
		insn       Instruction
	}

	var intervals []iv
	for _, r := range raw {
		m := mapOf(r)
		start, _ := asInt(get(m, "start"))
		end, _ := asInt(get(m, "end"))
		intervals = append(intervals, iv{start, end, routeInlineFormat(m)})
	}

	boundaries := map[int]bool{}
	for _, x := range intervals {
		boundaries[x.begin] = true
		boundaries[x.end] = true
	}
	coords := make([]int, 0, len(boundaries))
	for c := range boundaries {
		coords = append(coords, c)
	}
	sort.Ints(coords)

	var split []iv
	for _, x := range intervals {
		for i := 0; i+1 < len(coords); i++ {
			a, b := coords[i], coords[i+1]
			if a >= x.begin && b <= x.end && a < b {
				split = append(split, iv{a, b, x.insn})
			}
		}
	}
	sort.SliceStable(split, func(i, j int) bool {
		if split[i].begin != split[j].begin {
			return split[i].begin < split[j].begin
		}
		if split[i].end != split[j].end {
			return split[i].end < split[j].end
		}
		return split[i].insn.Type < split[j].insn.Type
	})

	type group struct {
		begin, end int
		insns      []Instruction
	}
	var discrete []*group
	var latch *group
	for _, x := range split {
		if latch != nil && latch.begin == x.begin && latch.end == x.end {
			latch.insns = append(latch.insns, x.insn)
			continue
		}
		if latch != nil {
			discrete = append(discrete, latch)
		}
		latch = &group{x.begin, x.end, []Instruction{x.insn}}
	}

	if len(discrete) == 0 {
		if latch != nil {
			discrete = append(discrete, latch)
		}
	} else {
		// Mirrors npf-renderer's slightly odd tail validation.
		lastRaw := split[len(split)-1]
		lastProc := discrete[len(discrete)-1]
		if lastRaw.begin != lastProc.begin && lastRaw.end != lastProc.end {
			if latch != nil {
				discrete = append(discrete, latch)
			}
		}
	}

	out := make([]StyleInterval, 0, len(discrete))
	for _, g := range discrete {
		insns := append([]Instruction(nil), g.insns...)
		sort.SliceStable(insns, func(i, j int) bool { return insns[i].Type < insns[j].Type })
		out = append(out, StyleInterval{Start: g.begin, End: g.end, Instructions: insns})
	}
	return out
}

func routeInlineFormat(m map[string]any) Instruction {
	typ := parseFMT(str(get(m, "type")))
	insn := Instruction{Type: typ}
	switch typ {
	case FMTLink:
		insn.URL = str(get(m, "url"))
	case FMTMention:
		blog := mapOf(get(m, "blog"))
		insn.BlogName = str(get(blog, "name"))
		insn.BlogUUID = str(get(blog, "uuid"))
		insn.BlogURL = str(get(blog, "url"))
	case FMTColor:
		insn.Hex = str(get(m, "hex"))
	}
	return insn
}

// ---------------------------------------------------------------------------
// Inline formatter
// ---------------------------------------------------------------------------

type opsIter struct {
	ops                []StyleInterval
	cursor             int
	current            *StyleInterval
	nextStart, nextEnd int
	hasNext            bool
}

func newOpsIter(ops []StyleInterval) *opsIter {
	o := &opsIter{ops: ops}
	o.updateNext()
	return o
}

func (o *opsIter) updateNext() {
	if o.cursor < len(o.ops) {
		o.nextStart = o.ops[o.cursor].Start
		o.nextEnd = o.ops[o.cursor].End
		o.hasNext = true
	} else {
		o.hasNext = false
	}
}

func (o *opsIter) next() bool {
	if o.cursor >= len(o.ops) {
		o.current = nil
		return false
	}
	o.current = &o.ops[o.cursor]
	o.cursor++
	o.updateNext()
	return true
}

type inlineFormatter struct {
	text       string
	cursor     int
	acc        []byte
	parent     *Node
	ops        *opsIter
	urlHandler func(string) string
}

func newInlineFormatter(text string, formats []StyleInterval, urlHandler func(string) string) *inlineFormatter {
	return &inlineFormatter{
		text:       text,
		parent:     El("span", "class", "inline-formatted-content"),
		ops:        newOpsIter(formats),
		urlHandler: urlHandler,
	}
}

func (f *inlineFormatter) atEnd() bool { return f.cursor >= len(f.text) }

func (f *inlineFormatter) next() bool {
	if f.cursor >= len(f.text) {
		return false
	}
	f.acc = append(f.acc, f.text[f.cursor])
	f.cursor++
	return true
}

func (f *inlineFormatter) dump(tag *Node) {
	tag.Add(Txt(string(f.acc)))
	f.acc = f.acc[:0]
}

func (f *inlineFormatter) makeTag(insn Instruction) *Node {
	switch insn.Type {
	case FMTBold:
		return El("b", "class", "inline-bold")
	case FMTItalic:
		return El("i", "class", "inline-italics")
	case FMTStrikethrough:
		return El("s", "class", "inline-strikethrough")
	case FMTSmall:
		return El("small", "class", "inline-small")
	case FMTColor:
		return El("span", "style", "color: "+insn.Hex+";", "class", "inline-color")
	case FMTLink:
		return El("a", "href", f.urlHandler(insn.URL), "class", "inline-link")
	case FMTMention:
		return El("a", "href", f.urlHandler(insn.BlogURL), "class", "inline-mention")
	}
	return El("span")
}

func (f *inlineFormatter) calcTags(op *StyleInterval) (*Node, *Node) {
	if len(op.Instructions) == 1 {
		tag := f.makeTag(op.Instructions[0])
		return tag, tag
	}
	tags := make([]*Node, 0, len(op.Instructions))
	for _, insn := range op.Instructions {
		tags = append(tags, f.makeTag(insn))
	}
	for i := 0; i+1 < len(tags); i++ {
		tags[i].Add(tags[i+1])
	}
	return tags[len(tags)-1], tags[0]
}

func (f *inlineFormatter) format() *Node {
	for !f.atEnd() {
		for f.ops.hasNext && f.cursor == f.ops.nextStart {
			f.ops.next()
			f.dump(f.parent)

			working, root := f.calcTags(f.ops.current)
			for f.cursor != f.ops.current.End && !f.atEnd() {
				f.next()
			}
			f.dump(working)
			f.parent.Add(root)
		}
		f.next()
	}
	f.dump(f.parent)
	return f.parent
}

// ---------------------------------------------------------------------------
// Text block formatter
// ---------------------------------------------------------------------------

func createTextTag(block *TextBlock, additional string) *Node {
	switch block.Subtype {
	case SubtypeNone:
		return El("p", "class", "text-block")
	case Heading1:
		return El("h1", "class", "text-block heading1"+additional)
	case Heading2:
		return El("h2", "class", "text-block heading2"+additional)
	case Quirky:
		return El("p", "class", "text-block quirky"+additional)
	case Quote:
		return El("blockquote", "class", "text-block quote"+additional)
	case Indented:
		return El("blockquote", "class", "text-block indented"+additional)
	case Chat:
		return El("p", "class", "text-block chat"+additional)
	case OrderedListItem:
		return El("li", "class", "text-block ordered-list-item"+additional)
	case UnorderedListItem:
		return El("li", "class", "text-block unordered-list-item"+additional)
	}
	return El("p", "class", "text-block")
}

// formatText renders a TextBlock (or ListGrouping) into HTML.
func formatText(block *TextBlock, urlHandler func(string) string) *Node {
	var textTag *Node
	additional := ""
	if len(block.InlineFormatting) > 0 {
		textTag = newInlineFormatter(block.Text, block.InlineFormatting, urlHandler).format()
		additional = " inline-formatted-block"
	} else {
		textTag = Txt(block.Text)
	}

	tag := createTextTag(block, additional)

	if len(block.Nest) > 0 {
		tag.Add(textTag)
		for _, element := range block.Nest {
			switch e := element.(type) {
			case *ListGrouping:
				listTag := listElement(e.Type)
				for _, b := range e.Group {
					listTag.Add(formatText(b, urlHandler))
				}
				tag.Add(listTag)
			case *TextBlock:
				tag.Add(formatText(e, urlHandler))
			}
		}
	} else {
		tag.Add(textTag)
	}
	return tag
}

func listElement(subtype Subtype) *Node {
	if subtype == OrderedListItem {
		return El("ol", "class", "ordered-list")
	}
	return El("ul", "class", "unordered-list")
}
