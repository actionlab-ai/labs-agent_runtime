package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

var knownIntentTerms = []string{
	"opening",
	"hook",
	"first chapter",
	"first-chapter",
	"novel",
	"urban",
	"power",
	"supernatural",
	"forensic",
	"report",
	"worldbuilding",
	"world",
	"setting",
	"idea",
	"premise",
	"cheat",
	"golden finger",
	"companion",
	"side character",
	"mission",
	"outline",
	"plot",
	"emotional core",
	"novel core",
	"story heart",
	"desire",
	"recognition",
	"catharsis",
	"\u5f00\u5934",
	"\u5f00\u7bc7",
	"\u7b2c\u4e00\u7ae0",
	"\u9ec4\u91d1600",
	"\u94a9\u5b50",
	"\u5f00\u5c40",
	"\u7f51\u6587",
	"\u5c0f\u8bf4",
	"\u90fd\u5e02",
	"\u5f02\u80fd",
	"\u5c38\u68c0",
	"\u62a5\u544a",
	"\u723d\u70b9",
	"\u60ac\u5ff5",
	"\u4e16\u754c\u89c2",
	"\u8bbe\u5b9a",
	"\u91d1\u624b\u6307",
	"\u521b\u610f",
	"\u8111\u6d1e",
	"\u8d77\u76d8",
	"\u4eba\u8bbe",
	"\u4e3b\u7ebf",
	"\u914d\u89d2",
	"\u60c5\u611f\u5185\u6838",
	"\u5c0f\u8bf4\u5185\u6838",
	"\u65b0\u4e66\u5185\u6838",
	"\u723d\u611f",
	"\u8ba4\u53ef\u611f",
	"\u4ee3\u507f",
	"\u538b\u8feb",
	"\u6e34\u671b",
}

var openingIntentTerms = []string{
	"opening",
	"hook",
	"first chapter",
	"first-chapter",
	"\u5f00\u5934",
	"\u5f00\u7bc7",
	"\u7b2c\u4e00\u7ae0",
	"\u9ec4\u91d1600",
	"\u94a9\u5b50",
	"\u5f00\u5c40",
}

type Command struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	WhenToUse       string         `json:"when_to_use"`
	Version         string         `json:"version,omitempty"`
	Tags            []string       `json:"tags,omitempty"`
	Aliases         []string       `json:"aliases,omitempty"`
	SearchHint      string         `json:"search_hint,omitempty"`
	AllowedTools    []string       `json:"allowed_tools,omitempty"`
	ArgumentHint    string         `json:"argument_hint,omitempty"`
	ToolDescription string         `json:"tool_description,omitempty"`
	ToolContract    string         `json:"tool_contract,omitempty"`
	ToolOutput      string         `json:"tool_output_contract,omitempty"`
	ToolInputSchema map[string]any `json:"tool_input_schema,omitempty"`
	Model           string         `json:"model,omitempty"`
	UserInvocable   bool           `json:"user_invocable"`
	EntryPath       string         `json:"entry_path"`
	SkillRoot       string         `json:"skill_root"`
	ContentLength   int            `json:"content_length"`
	MarkdownContent string         `json:"-"`
}

type Registry struct {
	SkillsDir string
	Commands  map[string]Command
}

type QueryExplanation struct {
	Raw           string   `json:"raw"`
	Mode          string   `json:"mode"`
	RequiredTerms []string `json:"required_terms,omitempty"`
	OptionalTerms []string `json:"optional_terms,omitempty"`
	ScoringTerms  []string `json:"scoring_terms,omitempty"`
}

type SearchHit struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	WhenToUse     string   `json:"when_to_use,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Score         float64  `json:"score"`
	Reason        string   `json:"reason"`
	MatchedFields []string `json:"matched_fields,omitempty"`
	MatchedTerms  []string `json:"matched_terms,omitempty"`
	Exact         bool     `json:"exact,omitempty"`
}

type searchQuery struct {
	QueryExplanation
	SelectedNames []string
}

type searchableCommand struct {
	IDParts    []string
	NameParts  []string
	AliasParts []string
	TagParts   []string
	Full       string
	Hint       string
	Narrative  string
}

func LoadRegistry(skillsDir string) (*Registry, error) {
	reg := &Registry{SkillsDir: skillsDir, Commands: map[string]Command{}}
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillID := e.Name()
		skillRoot := filepath.Join(skillsDir, skillID)
		entryPath := filepath.Join(skillRoot, "SKILL.md")
		data, firstParagraph, bodyLength, err := loadSkillMetadata(entryPath)
		if err != nil {
			return nil, fmt.Errorf("load skill metadata %s: %w", entryPath, err)
		}
		cmd := Command{
			ID:           skillID,
			Name:         firstNonEmpty(fmString(data, "name"), skillID),
			Description:  firstNonEmpty(fmString(data, "description"), firstParagraph),
			WhenToUse:    fmString(data, "when_to_use"),
			Version:      fmString(data, "version"),
			Tags:         fmStringSlice(data, "tags"),
			Aliases:      fmStringSlice(data, "aliases"),
			SearchHint:   fmString(data, "search_hint"),
			AllowedTools: fmStringSlice(data, "allowed_tools"),
			ArgumentHint: fmString(data, "argument_hint"),
			ToolDescription: firstNonEmpty(
				fmString(data, "tool_description"),
				fmString(data, "tool_prompt"),
			),
			ToolContract:    fmString(data, "tool_contract"),
			ToolOutput:      fmString(data, "tool_output_contract"),
			ToolInputSchema: fmMap(data, "tool_input_schema"),
			Model:           fmString(data, "model"),
			UserInvocable:   true,
			EntryPath:       entryPath,
			SkillRoot:       skillRoot,
			ContentLength:   bodyLength,
		}
		if v := fmString(data, "user_invocable"); strings.EqualFold(v, "false") {
			cmd.UserInvocable = false
		}
		reg.Commands[cmd.ID] = cmd
	}
	return reg, nil
}

func (r *Registry) List() []Command {
	out := make([]Command, 0, len(r.Commands))
	for _, c := range r.Commands {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (r *Registry) Get(id string) (Command, bool) {
	c, ok := r.Commands[id]
	return c, ok
}

func (r *Registry) LoadInvocationCommand(id string) (Command, error) {
	cmd, ok := r.Get(id)
	if !ok {
		return Command{}, fmt.Errorf("skill not found: %s", id)
	}
	body, err := loadSkillBody(cmd.EntryPath)
	if err != nil {
		return Command{}, fmt.Errorf("load skill body %s: %w", cmd.EntryPath, err)
	}
	cmd.MarkdownContent = body
	if cmd.ContentLength == 0 {
		cmd.ContentLength = len(body)
	}
	if strings.TrimSpace(cmd.Description) == "" {
		cmd.Description = extractFirstParagraph(body)
	}
	return cmd, nil
}

func ExplainQuery(query string) QueryExplanation {
	parsed := parseSearchQuery(query)
	return parsed.QueryExplanation
}

func (r *Registry) Search(query string, limit int) []SearchHit {
	if limit <= 0 {
		limit = 5
	}
	parsed := parseSearchQuery(query)
	if len(parsed.SelectedNames) > 0 {
		return r.searchSelected(parsed.SelectedNames, limit)
	}
	if exact, ok := r.searchExact(parsed.Raw); ok {
		return []SearchHit{exact}
	}

	var hits []SearchHit
	for _, c := range r.Commands {
		if !c.UserInvocable {
			continue
		}
		index := buildSearchableCommand(c)
		if !matchesRequiredTerms(parsed.RequiredTerms, index) {
			continue
		}
		score, fields, terms := scoreCommand(parsed, c, index)
		if score <= 0 {
			continue
		}
		hits = append(hits, SearchHit{
			ID:            c.ID,
			Name:          c.Name,
			Description:   c.Description,
			WhenToUse:     c.WhenToUse,
			Tags:          c.Tags,
			Score:         score,
			Reason:        buildReason(fields, terms),
			MatchedFields: fields,
			MatchedTerms:  terms,
		})
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			return hits[i].ID < hits[j].ID
		}
		return hits[i].Score > hits[j].Score
	})
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

func parseSearchQuery(query string) searchQuery {
	raw := strings.TrimSpace(query)
	if strings.HasPrefix(strings.ToLower(raw), "select:") {
		parts := strings.Split(strings.TrimSpace(raw[len("select:"):]), ",")
		var names []string
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				names = append(names, part)
			}
		}
		return searchQuery{
			QueryExplanation: QueryExplanation{
				Raw:  raw,
				Mode: "select",
			},
			SelectedNames: unique(names),
		}
	}

	allTerms := tokenizeQuery(raw)
	var required []string
	var optional []string
	requiredSet := map[string]bool{}

	for _, field := range strings.Fields(normalizeSearchText(raw)) {
		if strings.HasPrefix(field, "+") && len(field) > 1 {
			term := strings.TrimPrefix(field, "+")
			required = append(required, term)
			requiredSet[term] = true
		}
	}
	required = unique(required)

	for _, term := range allTerms {
		if !requiredSet[term] {
			optional = append(optional, term)
		}
	}
	optional = unique(optional)

	scoringTerms := optional
	if len(required) > 0 {
		scoringTerms = append(append([]string{}, required...), optional...)
	}

	return searchQuery{
		QueryExplanation: QueryExplanation{
			Raw:           raw,
			Mode:          "keyword",
			RequiredTerms: required,
			OptionalTerms: optional,
			ScoringTerms:  unique(scoringTerms),
		},
	}
}

func (r *Registry) searchSelected(selected []string, limit int) []SearchHit {
	var hits []SearchHit
	for _, name := range selected {
		cmd, reason, ok := r.findSelectable(name)
		if !ok {
			continue
		}
		hits = append(hits, SearchHit{
			ID:            cmd.ID,
			Name:          cmd.Name,
			Description:   cmd.Description,
			WhenToUse:     cmd.WhenToUse,
			Tags:          cmd.Tags,
			Score:         2.0,
			Reason:        reason,
			MatchedFields: []string{"select"},
			MatchedTerms:  []string{strings.TrimSpace(name)},
			Exact:         true,
		})
	}
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

func (r *Registry) searchExact(raw string) (SearchHit, bool) {
	target := normalizeIdentity(raw)
	if target == "" {
		return SearchHit{}, false
	}
	for _, c := range r.Commands {
		if !c.UserInvocable {
			continue
		}
		if normalizeIdentity(c.ID) == target {
			return exactHit(c, "exact-id", raw), true
		}
		if normalizeIdentity(c.Name) == target {
			return exactHit(c, "exact-name", raw), true
		}
		for _, alias := range c.Aliases {
			if normalizeIdentity(alias) == target {
				return exactHit(c, "exact-alias", raw), true
			}
		}
	}
	return SearchHit{}, false
}

func exactHit(c Command, field, raw string) SearchHit {
	return SearchHit{
		ID:            c.ID,
		Name:          c.Name,
		Description:   c.Description,
		WhenToUse:     c.WhenToUse,
		Tags:          c.Tags,
		Score:         2.2,
		Reason:        fmt.Sprintf("fields: %s; terms: %s", field, strings.TrimSpace(raw)),
		MatchedFields: []string{field},
		MatchedTerms:  []string{strings.TrimSpace(raw)},
		Exact:         true,
	}
}

func (r *Registry) findSelectable(name string) (Command, string, bool) {
	target := strings.TrimSpace(name)
	if target == "" {
		return Command{}, "", false
	}
	for _, c := range r.Commands {
		if !c.UserInvocable {
			continue
		}
		switch {
		case strings.EqualFold(c.ID, target):
			return c, "fields: select-exact-id", true
		case strings.EqualFold(c.Name, target):
			return c, "fields: select-exact-name", true
		}
		for _, alias := range c.Aliases {
			if strings.EqualFold(alias, target) {
				return c, "fields: select-alias", true
			}
		}
	}
	return Command{}, "", false
}

func scoreCommand(q searchQuery, c Command, index searchableCommand) (float64, []string, []string) {
	total := 0.0
	var matchedFields []string
	var matchedTerms []string

	for _, term := range q.ScoringTerms {
		score, fields := scoreTerm(term, index)
		if score <= 0 {
			continue
		}
		total += score
		matchedFields = append(matchedFields, fields...)
		matchedTerms = append(matchedTerms, term)
	}

	matchedFields = unique(matchedFields)
	matchedTerms = unique(matchedTerms)
	if len(matchedTerms) == 0 {
		return 0, nil, nil
	}

	if containsIntentTerm(q.Raw, openingIntentTerms) && containsIntentTerm(commandIntentText(c, index), openingIntentTerms) {
		total += 3
		matchedFields = append(matchedFields, "opening-intent")
	}

	final := total / float64(len(q.ScoringTerms)+3)
	return final, unique(matchedFields), matchedTerms
}

func matchesRequiredTerms(requiredTerms []string, index searchableCommand) bool {
	if len(requiredTerms) == 0 {
		return true
	}
	for _, term := range requiredTerms {
		if !matchesTerm(term, index) {
			return false
		}
	}
	return true
}

func matchesTerm(term string, index searchableCommand) bool {
	return containsExactPart(index.IDParts, term) ||
		containsExactPart(index.NameParts, term) ||
		containsExactPart(index.AliasParts, term) ||
		containsExactPart(index.TagParts, term) ||
		containsPartialPart(index.IDParts, term) ||
		containsPartialPart(index.NameParts, term) ||
		containsPartialPart(index.AliasParts, term) ||
		containsPartialPart(index.TagParts, term) ||
		textContainsToken(index.Hint, term) ||
		textContainsToken(index.Narrative, term)
}

func scoreTerm(term string, index searchableCommand) (float64, []string) {
	var score float64
	var fields []string

	switch {
	case containsExactPart(index.IDParts, term):
		score += 12
		fields = append(fields, "id")
	case containsPartialPart(index.IDParts, term):
		score += 6
		fields = append(fields, "id-partial")
	}

	switch {
	case containsExactPart(index.NameParts, term):
		score += 11
		fields = append(fields, "name")
	case containsPartialPart(index.NameParts, term):
		score += 5
		fields = append(fields, "name-partial")
	}

	switch {
	case containsExactPart(index.AliasParts, term):
		score += 10
		fields = append(fields, "alias")
	case containsPartialPart(index.AliasParts, term):
		score += 5
		fields = append(fields, "alias-partial")
	}

	switch {
	case containsExactPart(index.TagParts, term):
		score += 10
		fields = append(fields, "tag")
	case containsPartialPart(index.TagParts, term):
		score += 5
		fields = append(fields, "tag-partial")
	}

	if score == 0 && strings.Contains(index.Full, term) {
		score += 3
		fields = append(fields, "full-name")
	}
	if textContainsToken(index.Hint, term) {
		score += 4
		fields = append(fields, "search-hint")
	}
	if textContainsToken(index.Narrative, term) {
		score += 2
		fields = append(fields, "description")
	}

	return score, unique(fields)
}

func buildSearchableCommand(c Command) searchableCommand {
	idParts := parseSearchParts(c.ID)
	nameParts := parseSearchParts(c.Name)
	var aliasParts []string
	for _, alias := range c.Aliases {
		aliasParts = append(aliasParts, parseSearchParts(alias)...)
	}
	var tagParts []string
	for _, tag := range c.Tags {
		tagParts = append(tagParts, parseSearchParts(tag)...)
	}

	full := normalizeSearchText(strings.Join([]string{
		c.ID,
		c.Name,
		strings.Join(c.Aliases, " "),
		strings.Join(c.Tags, " "),
	}, " "))

	return searchableCommand{
		IDParts:    unique(idParts),
		NameParts:  unique(nameParts),
		AliasParts: unique(aliasParts),
		TagParts:   unique(tagParts),
		Full:       full,
		Hint:       normalizeSearchText(c.SearchHint),
		Narrative:  normalizeSearchText(strings.Join([]string{c.WhenToUse, c.Description}, " ")),
	}
}

func parseSearchParts(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	return tokenizeQuery(insertCamelBreaks(text))
}

func insertCamelBreaks(text string) string {
	var out []rune
	prevLowerOrDigit := false
	for _, r := range text {
		if prevLowerOrDigit && unicode.IsUpper(r) {
			out = append(out, ' ')
		}
		out = append(out, r)
		prevLowerOrDigit = unicode.IsLower(r) || unicode.IsDigit(r)
	}
	return string(out)
}

func tokenizeQuery(text string) []string {
	normalized := normalizeSearchText(text)
	fields := strings.Fields(normalized)
	var out []string
	for _, field := range fields {
		field = strings.TrimSpace(strings.TrimPrefix(field, "+"))
		if field == "" {
			continue
		}
		out = append(out, field)
	}
	for _, term := range knownIntentTerms {
		normalizedTerm := normalizeSearchText(term)
		if normalizedTerm != "" && strings.Contains(normalized, normalizedTerm) {
			out = append(out, normalizedTerm)
		}
	}
	return unique(out)
}

func normalizeSearchText(s string) string {
	s = strings.ToLower(s)
	replacer := strings.NewReplacer(
		"\u3000", " ",
		"\u3001", " ",
		"\u3002", " ",
		"\uFF0C", " ",
		"\uFF1A", " ",
		"\uFF1B", " ",
		",", " ",
		".", " ",
		":", " ",
		";", " ",
		"-", " ",
		"_", " ",
		"/", " ",
		"\n", " ",
		"\t", " ",
	)
	return replacer.Replace(s)
}

func normalizeIdentity(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func buildReason(fields, terms []string) string {
	switch {
	case len(fields) == 0 && len(terms) == 0:
		return "matched"
	case len(fields) == 0:
		return "terms: " + strings.Join(terms, ", ")
	case len(terms) == 0:
		return "fields: " + strings.Join(fields, ", ")
	default:
		return "fields: " + strings.Join(fields, ", ") + "; terms: " + strings.Join(terms, ", ")
	}
}

func commandIntentText(c Command, index searchableCommand) string {
	return strings.Join([]string{
		c.ID,
		c.Name,
		strings.Join(c.Tags, " "),
		c.SearchHint,
		index.Narrative,
	}, " ")
}

func containsIntentTerm(text string, terms []string) bool {
	normalized := normalizeSearchText(text)
	for _, term := range terms {
		needle := normalizeSearchText(term)
		if needle != "" && strings.Contains(normalized, needle) {
			return true
		}
	}
	return false
}

func containsExactPart(parts []string, term string) bool {
	term = normalizeSearchText(term)
	for _, part := range parts {
		if normalizeSearchText(part) == term {
			return true
		}
	}
	return false
}

func containsPartialPart(parts []string, term string) bool {
	term = normalizeSearchText(term)
	if term == "" {
		return false
	}
	for _, part := range parts {
		part = normalizeSearchText(part)
		switch {
		case len(term) >= 2 && strings.Contains(part, term):
			return true
		case len(part) >= 3 && strings.Contains(term, part):
			return true
		}
	}
	return false
}

func textContainsToken(text, term string) bool {
	text = normalizeSearchText(text)
	term = normalizeSearchText(term)
	if text == "" || term == "" {
		return false
	}
	if !looksASCIIWord(term) {
		return strings.Contains(text, term)
	}
	for _, token := range strings.Fields(text) {
		if token == term {
			return true
		}
	}
	return strings.Contains(text, term)
}

func looksASCIIWord(term string) bool {
	for _, r := range term {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func CloneMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		switch x := v.(type) {
		case map[string]any:
			out[k] = CloneMap(x)
		case []any:
			out[k] = cloneSlice(x)
		default:
			out[k] = x
		}
	}
	return out
}

func cloneSlice(src []any) []any {
	if len(src) == 0 {
		return nil
	}
	out := make([]any, len(src))
	for i, v := range src {
		switch x := v.(type) {
		case map[string]any:
			out[i] = CloneMap(x)
		case []any:
			out[i] = cloneSlice(x)
		default:
			out[i] = x
		}
	}
	return out
}

func extractFirstParagraph(s string) string {
	parts := strings.Split(s, "\n\n")
	for _, p := range parts {
		p = strings.TrimSpace(strings.ReplaceAll(p, "#", ""))
		if p != "" {
			if len(p) > 220 {
				return p[:220]
			}
			return p
		}
	}
	return ""
}

func unique(xs []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range xs {
		x = strings.TrimSpace(x)
		if x == "" || seen[x] {
			continue
		}
		seen[x] = true
		out = append(out, x)
	}
	return out
}
