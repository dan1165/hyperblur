package npf

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// PollCallback fetches results for a poll. endTimestamp is the poll's
// expiration time (creation + expires_after).
type PollCallback func(pollID string, endTimestamp int64) (map[string]PollResult, error)

// ---------------------------------------------------------------------------
// JSON value helpers
// ---------------------------------------------------------------------------

func get(m map[string]any, key string) any {
	if m == nil {
		return nil
	}
	return m[key]
}

func getOr(m map[string]any, keys ...string) any {
	for _, k := range keys {
		if v := get(m, k); v != nil {
			return v
		}
	}
	return nil
}

func mapOf(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asString(v any) (string, bool) {
	s, ok := v.(string)
	return s, ok
}

func str(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func asInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	}
	return 0, false
}

func asInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int:
		return int64(n), true
	case int64:
		return n, true
	}
	return 0, false
}

func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0
	case int:
		return t != 0
	case int64:
		return t != 0
	case []any:
		return len(t) > 0
	case map[string]any:
		return len(t) > 0
	}
	return true
}

func indentLevel(m map[string]any) (int, bool) {
	for _, k := range []string{"indent_level", "indentLevel"} {
		if v, ok := m[k]; ok && v != nil {
			if n, ok := asInt(v); ok {
				return n, true
			}
			return 0, true
		}
	}
	return 0, false
}

func parseSubtype(s string) Subtype {
	switch strings.ToUpper(strings.ReplaceAll(s, "-", "_")) {
	case "HEADING1":
		return Heading1
	case "HEADING2":
		return Heading2
	case "QUIRKY":
		return Quirky
	case "QUOTE":
		return Quote
	case "INDENTED":
		return Indented
	case "CHAT":
		return Chat
	case "ORDERED_LIST_ITEM":
		return OrderedListItem
	case "UNORDERED_LIST_ITEM":
		return UnorderedListItem
	}
	return SubtypeNone
}

func parseFMT(s string) FMTType {
	switch strings.ToUpper(s) {
	case "BOLD":
		return FMTBold
	case "ITALIC":
		return FMTItalic
	case "STRIKETHROUGH":
		return FMTStrikethrough
	case "SMALL":
		return FMTSmall
	case "LINK":
		return FMTLink
	case "MENTION":
		return FMTMention
	case "COLOR":
		return FMTColor
	}
	return FMTBold
}

// ---------------------------------------------------------------------------
// Media / attribution parsing
// ---------------------------------------------------------------------------

func parseMediaObjects(raw any) []MediaObject {
	if raw == nil {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		list = []any{raw}
	}
	var out []MediaObject
	for _, item := range list {
		if m, ok := item.(map[string]any); ok {
			out = append(out, parseMediaBlock(m))
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseMediaBlock(m map[string]any) MediaObject {
	mo := MediaObject{URL: str(get(m, "url")), Width: 540, Height: 405}
	if t, ok := asString(get(m, "type")); ok {
		mo.Type = t
	}
	if w, ok := asInt(get(m, "width")); ok {
		mo.Width = w
	}
	if h, ok := asInt(get(m, "height")); ok {
		mo.Height = h
	}
	mo.Cropped = truthy(get(m, "cropped"))

	hod := get(m, "has_original_dimensions")
	if !truthy(hod) {
		hod = get(m, "hasOriginalDimensions")
	}
	mo.HasOriginalDimensions = truthy(hod)

	return mo
}

func parseAttribution(m map[string]any) Attribution {
	switch str(get(m, "type")) {
	case "post":
		blog := mapOf(get(m, "blog"))
		return Attribution{
			Kind: AttrPost,
			URL:  str(get(m, "url")),
			Blog: BlogAttribution{
				URL:  str(get(blog, "url")),
				Name: str(get(blog, "name")),
			},
		}
	case "link":
		return Attribution{Kind: AttrLink, URL: str(get(m, "url"))}
	case "blog":
		blog := mapOf(get(m, "blog"))
		var avatars []MediaObject
		if av := get(blog, "avatar"); av != nil {
			if l, ok := av.([]any); ok {
				for _, a := range l {
					if am, ok := a.(map[string]any); ok {
						avatars = append(avatars, parseMediaBlock(am))
					}
				}
			}
		}
		return Attribution{
			Kind: AttrBlog,
			URL:  str(get(m, "url")),
			Blog: BlogAttribution{
				Name:   str(get(blog, "name")),
				Avatar: avatars,
			},
		}
	case "app":
		name := str(get(m, "app_name"))
		if name == "" {
			name = str(get(m, "appName"))
		}
		return Attribution{
			Kind:    AttrApp,
			URL:     str(get(m, "url")),
			AppName: name,
		}
	default:
		return Attribution{Kind: AttrUnsupported, TypeStr: str(get(m, "type"))}
	}
}

// ---------------------------------------------------------------------------
// Parser
// ---------------------------------------------------------------------------

// Parser converts NPF content blocks into parsed objects.
type Parser struct {
	content      []map[string]any
	idx          int
	current      map[string]any
	parsed       []any
	pollCallback PollCallback
}

// newParser builds a parser over raw NPF content blocks.
func newParser(content []any, cb PollCallback) *Parser {
	blocks := make([]map[string]any, 0, len(content))
	for _, c := range content {
		if m, ok := c.(map[string]any); ok {
			blocks = append(blocks, m)
		}
	}
	return &Parser{content: blocks, pollCallback: cb}
}

func (p *Parser) next() bool {
	if p.idx >= len(p.content) {
		p.current = nil
		return false
	}
	p.current = p.content[p.idx]
	p.idx++
	return true
}

func (p *Parser) peek() map[string]any {
	if p.idx >= len(p.content) {
		return nil
	}
	return p.content[p.idx]
}

// Parse runs the parser and returns the parsed top-level blocks.
func (p *Parser) Parse() ([]any, error) {
	for p.next() {
		if err := p.parseBlock(); err != nil {
			return nil, err
		}
	}
	return p.parsed, nil
}

func (p *Parser) parseBlock() error {
	switch str(get(p.current, "type")) {
	case "text":
		p.parsed = append(p.parsed, p.parseText(0, false))
	case "image":
		p.parsed = append(p.parsed, p.parseImageBlock())
	case "link":
		p.parsed = append(p.parsed, p.parseLinkBlock())
	case "audio":
		p.parsed = append(p.parsed, p.parseAudioBlock())
	case "video":
		p.parsed = append(p.parsed, p.parseVideoBlock())
	case "poll":
		b, err := p.parsePollBlock()
		if err != nil {
			return err
		}
		p.parsed = append(p.parsed, b)
	default:
		p.parsed = append(p.parsed, &Unsupported{Type: str(get(p.current, "type"))})
	}
	return nil
}

func (p *Parser) parseText(nestLevel int, inListGrouping bool) any {
	text := str(get(p.current, "text"))

	var subtype Subtype
	if s, ok := asString(get(p.current, "subtype")); ok && s != "" {
		subtype = parseSubtype(s)
	}

	var inlineFormats []StyleInterval
	if f := get(p.current, "formatting"); f != nil {
		if list, ok := f.([]any); ok {
			inlineFormats = p.parseInlineText(list)
		}
	}

	var nest []any
	for {
		peek := p.peek()
		if peek == nil {
			break
		}
		if str(get(peek, "type")) != "text" {
			break
		}
		level, has := indentLevel(peek)
		if !has {
			break
		}
		if level > nestLevel {
			p.next()
			nest = append(nest, p.parseText(nestLevel+1, false))
		} else {
			break
		}
	}

	if !inListGrouping && isListSubtype(subtype) {
		group := []*TextBlock{newTextBlock(text, subtype, inlineFormats, nest)}
		for {
			peek := p.peek()
			if peek == nil {
				break
			}
			var peekSubtype Subtype
			if s, ok := asString(get(peek, "subtype")); ok && s != "" {
				peekSubtype = parseSubtype(s)
			}
			level, _ := indentLevel(peek)
			if peekSubtype != subtype || level != nestLevel {
				break
			}
			p.next()
			item, _ := p.parseText(nestLevel, true).(*TextBlock)
			group = append(group, item)
		}
		return &ListGrouping{Type: subtype, Group: group}
	}

	return newTextBlock(text, subtype, inlineFormats, nest)
}

func newTextBlock(text string, subtype Subtype, inline []StyleInterval, nest []any) *TextBlock {
	block := &TextBlock{Text: text, Subtype: subtype, InlineFormatting: inline}
	if len(nest) > 0 {
		block.Nest = nest
	}
	return block
}

func (p *Parser) parseImageBlock() *ImageBlock {
	alt, _ := asString(getOr(p.current, "alt_text", "altText"))
	caption, _ := asString(get(p.current, "caption"))

	var attr *Attribution
	if a, ok := get(p.current, "attribution").(map[string]any); ok {
		parsed := parseAttribution(a)
		attr = &parsed
	}

	return &ImageBlock{
		Media:       parseMediaObjects(get(p.current, "media")),
		AltText:     alt,
		Caption:     caption,
		Attribution: attr,
	}
}

func (p *Parser) parseLinkBlock() *LinkBlock {
	block := &LinkBlock{
		URL:    str(get(p.current, "url")),
		Poster: parseMediaObjects(get(p.current, "poster")),
	}
	block.Title, _ = asString(get(p.current, "title"))
	block.Description, _ = asString(get(p.current, "description"))
	block.Author, _ = asString(get(p.current, "author"))
	block.SiteName, _ = asString(getOr(p.current, "siteName", "site_name"))
	return block
}

func (p *Parser) fetchAudiovisual() (string, string, []MediaObject, []MediaObject, string, string, *EmbedIframe) {
	url := str(get(p.current, "url"))
	provider := str(get(p.current, "provider"))
	media := parseMediaObjects(get(p.current, "media"))

	embedHTML, _ := asString(getOr(p.current, "embedHtml", "embed_html"))
	embedURL, _ := asString(getOr(p.current, "embedUrl", "embed_url"))

	var embedIframe *EmbedIframe
	if ei, ok := getOr(p.current, "embedIframe", "embed_iframe").(map[string]any); ok {
		iframe := &EmbedIframe{URL: str(get(ei, "url")), Width: 540, Height: 405}
		if w, ok := asInt(get(ei, "width")); ok {
			iframe.Width = w
		}
		if h, ok := asInt(get(ei, "height")); ok {
			iframe.Height = h
		}
		embedIframe = iframe
	}

	poster := parseMediaObjects(get(p.current, "poster"))

	return url, provider, media, poster, embedHTML, embedURL, embedIframe
}

func (p *Parser) parseVideoBlock() *VideoBlock {
	url, provider, media, poster, embedHTML, embedURL, embedIframe := p.fetchAudiovisual()
	return &VideoBlock{
		URL:         url,
		Provider:    provider,
		Media:       media,
		EmbedHTML:   embedHTML,
		EmbedIframe: embedIframe,
		EmbedURL:    embedURL,
		Poster:      poster,
	}
}

func (p *Parser) parseAudioBlock() *AudioBlock {
	url, provider, media, poster, embedHTML, embedURL, _ := p.fetchAudiovisual()
	return &AudioBlock{
		URL:       url,
		Provider:  provider,
		Media:     media,
		Poster:    poster,
		EmbedHTML: embedHTML,
		EmbedURL:  embedURL,
		Title:     str(get(p.current, "title")),
		Artist:    str(get(p.current, "artist")),
		Album:     str(get(p.current, "album")),
	}
}

func (p *Parser) parsePollBlock() (*PollBlock, error) {
	pollID := getOr(p.current, "clientId", "client_id")
	if pollID == nil {
		return nil, errors.New("invalid poll ID")
	}

	answersRaw, _ := get(p.current, "answers").([]any)
	answers := make([]Answer, 0, len(answersRaw))
	for _, raw := range answersRaw {
		am := mapOf(raw)
		id := getOr(am, "clientId", "client_id")
		text := getOr(am, "answerText", "answer_text")
		if id == nil || text == nil {
			return nil, errors.New("invalid poll answer")
		}
		answers = append(answers, Answer{ID: str(id), Text: str(text)})
	}

	creation, _ := asInt64(get(p.current, "timestamp"))
	settings := mapOf(get(p.current, "settings"))
	expiresAfter, _ := asInt64(get(settings, "expireAfter"))

	var votes map[string]PollResult
	totalVotes := 0
	if p.pollCallback != nil {
		results, err := p.pollCallback(str(pollID), creation+expiresAfter)
		if err != nil {
			return nil, err
		}
		if results != nil {
			type pair struct {
				id    string
				count int
			}
			pairs := make([]pair, 0, len(results))
			for id, r := range results {
				pairs = append(pairs, pair{id, r.VoteCount})
			}
			sort.SliceStable(pairs, func(i, j int) bool { return pairs[i].count > pairs[j].count })

			votes = map[string]PollResult{}
			for i, pr := range pairs {
				totalVotes += pr.count
				votes[pr.id] = PollResult{IsWinner: i == 0, VoteCount: pr.count}
			}
		}
	}

	return &PollBlock{
		PollID:            str(pollID),
		Question:          str(get(p.current, "question")),
		Answers:           answers,
		CreationTimestamp: creation,
		ExpiresAfter:      expiresAfter,
		Votes:             votes,
		TotalVotes:        totalVotes,
	}, nil
}

// ---------------------------------------------------------------------------
// Layout parser
// ---------------------------------------------------------------------------

// LayoutParser parses NPF layout information.
type LayoutParser struct {
	layouts    []map[string]any
	idx        int
	current    map[string]any
	askIndices map[int]bool
	result     []any
}

// newLayoutParser builds a parser over raw layout blocks.
func newLayoutParser(layouts []any) *LayoutParser {
	blocks := make([]map[string]any, 0, len(layouts))
	for _, l := range layouts {
		if m, ok := l.(map[string]any); ok {
			blocks = append(blocks, m)
		}
	}
	return &LayoutParser{layouts: blocks, askIndices: map[int]bool{}}
}

func (p *LayoutParser) next() bool {
	if p.idx >= len(p.layouts) {
		p.current = nil
		return false
	}
	p.current = p.layouts[p.idx]
	p.idx++
	return true
}

func intList(v any) []int {
	list, _ := v.([]any)
	out := make([]int, 0, len(list))
	for _, item := range list {
		if n, ok := asInt(item); ok {
			out = append(out, n)
		}
	}
	return out
}

// Parse returns the parsed layout blocks.
func (p *LayoutParser) Parse() []any {
	for p.next() {
		switch str(get(p.current, "type")) {
		case "ask":
			indices := intList(get(p.current, "blocks"))
			var attr *BlogAttribution
			if a, ok := get(p.current, "attribution").(map[string]any); ok {
				if parsed := parseAttribution(a); parsed.Kind == AttrBlog {
					attr = &parsed.Blog
				}
			}
			p.result = append(p.result, &AskLayout{Ranges: indices, Attribution: attr})
			for _, i := range indices {
				p.askIndices[i] = true
			}
		case "rows":
			var rows []RowLayout
			display, _ := get(p.current, "display").([]any)
			for _, raw := range display {
				row := mapOf(raw)
				var indices []int
				for _, i := range intList(get(row, "blocks")) {
					if !p.askIndices[i] {
						indices = append(indices, i)
					}
				}
				if len(indices) == 0 {
					continue
				}
				rows = append(rows, RowLayout{Ranges: indices})
			}
			p.result = append(p.result, &Rows{Rows: rows})
		}
	}
	return p.result
}
