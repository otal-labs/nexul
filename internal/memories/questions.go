package memories

import (
	"fmt"
	"strings"
	"unicode/utf8"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// MaxInterviewTemplateChars is the Interview template's storage limit; the template is never sent with a turn.
const MaxInterviewTemplateChars = 32_000

// Question is one question of the Interview template; its trimmed text is its identity.
type Question struct {
	Text        string   `json:"text"`
	Hint        string   `json:"hint"`
	MultiSelect bool     `json:"multi_select"`
	Options     []Option `json:"options"`
}

// Option is one choice a question offers; free text is always offered beside them.
type Option struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// ParseInterviewTemplate reads the template's questions: a `##` heading each, a hint up to the first bullet, then
// `- ` single-choice or `- [ ]` multi-select options. Questions come back with the first error, if any.
func ParseInterviewTemplate(body string) ([]Question, error) {
	p := &templateParser{questions: []Question{}, seen: map[string]int{}}
	for i, line := range strings.Split(body, "\n") {
		p.line(i+1, strings.TrimSpace(line))
	}
	p.finish()
	if p.err == nil && len(p.questions) == 0 {
		p.err = fmt.Errorf("%w: the Interview template has no questions; start each with a ## heading", apperrs.ErrInvalid)
	}
	return p.questions, p.err
}

type templateParser struct {
	questions []Question
	current   *Question
	hint      []string
	bulletAt  int
	seen      map[string]int
	err       error
}

func (p *templateParser) fail(n int, format string, args ...any) {
	if p.err == nil {
		p.err = fmt.Errorf("%w: line %d: %s", apperrs.ErrInvalid, n, fmt.Sprintf(format, args...))
	}
}

func (p *templateParser) line(n int, line string) {
	if line == "##" || strings.HasPrefix(line, "## ") {
		p.heading(n, strings.TrimSpace(line[2:]))
		return
	}
	if line == "" {
		return
	}
	if p.current == nil {
		p.fail(n, "text before the first ## question heading")
		return
	}
	if line == "-" || strings.HasPrefix(line, "- ") {
		p.option(n, strings.TrimSpace(line[1:]))
		return
	}
	if p.bulletAt == 0 {
		p.hint = append(p.hint, line)
		return
	}
	if last := len(p.current.Options) - 1; last >= 0 {
		p.current.Options[last].Description = strings.TrimSpace(p.current.Options[last].Description + " " + line)
	}
}

func (p *templateParser) heading(n int, text string) {
	p.finish()
	if text == "" {
		p.fail(n, "an empty ## heading; write the question after it")
		return
	}
	if first, ok := p.seen[text]; ok {
		p.fail(n, "the question %q is already asked on line %d", text, first)
	}
	p.seen[text] = n
	p.current = &Question{Text: text, Options: []Option{}}
}

func (p *templateParser) option(n int, text string) {
	multi := text == "[ ]" || strings.HasPrefix(text, "[ ] ")
	if multi {
		text = strings.TrimSpace(text[3:])
	}
	if p.bulletAt == 0 {
		p.bulletAt = n
		p.current.MultiSelect = multi
	}
	if multi != p.current.MultiSelect {
		p.fail(n, "%q mixes - [ ] multi-select options with - single-choice ones; use one kind per question", p.current.Text)
		return
	}
	label, description, _ := strings.Cut(text, ": ")
	label = strings.TrimSpace(label)
	if label == "" {
		return
	}
	p.current.Options = append(p.current.Options, Option{Label: label, Description: strings.TrimSpace(description)})
}

func (p *templateParser) finish() {
	if p.current == nil {
		return
	}
	p.current.Hint = strings.Join(p.hint, "\n")
	p.questions = append(p.questions, *p.current)
	p.current, p.hint, p.bulletAt = nil, nil, 0
}

// CheckInterviewTemplate refuses an Interview template over its storage limit or one the page cannot show as questions.
func CheckInterviewTemplate(body string) error {
	if n := utf8.RuneCountInString(body); n > MaxInterviewTemplateChars {
		return fmt.Errorf("%w: the Interview template is %d characters, over the %d-character limit", apperrs.ErrInvalid, n, MaxInterviewTemplateChars)
	}
	_, err := ParseInterviewTemplate(body)
	return err
}

// TemplateQuestions parses for display: a template stored before the format existed shows what it can.
func TemplateQuestions(body string) []Question {
	qs, _ := ParseInterviewTemplate(body) // the save path refuses; a read never fails on stored text
	return qs
}
