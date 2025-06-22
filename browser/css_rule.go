package browser

type Rule struct {
	Selector Selector
	Body     map[string]string
	Media    string
}

func NewRule(media string, selector Selector, body map[string]string) *Rule {
	return &Rule{
		Media:    media,
		Selector: selector,
		Body:     body,
	}
}

func CascadePriority(rule Rule) int {
	sel := rule.Selector
	return sel.Priority()
}
