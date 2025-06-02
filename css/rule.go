package css

type Rule struct {
	Selector Selector
	Body     map[string]string
}

func NewRule(selector Selector, body map[string]string) *Rule {
	return &Rule{
		Selector: selector,
		Body:     body,
	}
}

func CascadePriority(rule Rule) int {
	sel := rule.Selector
	return sel.Priority()
}
