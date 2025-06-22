package browser

import (
	"testing"
)

func TestSimpleSelector(t *testing.T) {
	tests := []struct {
		input         string
		wantType      string // "tag", "class", "pseudo"
		wantValue     string // tag/class/pseudo
		wantBaseValue string // for pseudo: base tag/class
	}{
		{"div", "tag", "div", ""},
		{".foo", "class", ".foo", ""},
		{"input:focus", "pseudo", "focus", "input"},
		{".bar:focus", "pseudo", "focus", ".bar"},
	}

	for _, tt := range tests {
		p := NewCSSParser(tt.input)
		sel := p.simple_selector()
		switch tt.wantType {
		case "tag":
			ts, ok := sel.(*TagSelector)
			if !ok {
				t.Errorf("input %q: expected tagSelector, got %T", tt.input, sel)
			} else if ts.Tag != tt.wantValue {
				t.Errorf("input %q: expected tag %q, got %q", tt.input, tt.wantValue, ts.Tag)
			}
		case "class":
			cs, ok := sel.(*ClassSelector)
			if !ok {
				t.Errorf("input %q: expected classSelector, got %T", tt.input, sel)
			} else if cs.Class != tt.wantValue {
				t.Errorf("input %q: expected class %q, got %q", tt.input, tt.wantValue, cs.Class)
			}
		case "pseudo":
			ps, ok := sel.(*PseudoclassSelector)
			if !ok {
				t.Errorf("input %q: expected pseudoclassSelector, got %T", tt.input, sel)
			} else if ps.pseudoclass != tt.wantValue {
				t.Errorf("input %q: expected pseudo %q, got %q", tt.input, tt.wantValue, ps.pseudoclass)
			} else {
				// Check base selector
				switch base := ps.base.(type) {
				case *TagSelector:
					if base.Tag != tt.wantBaseValue {
						t.Errorf("input %q: expected base tag %q, got %q", tt.input, tt.wantBaseValue, base.Tag)
					}
				case *ClassSelector:
					if base.Class != tt.wantBaseValue {
						t.Errorf("input %q: expected base class %q, got %q", tt.input, tt.wantBaseValue, base.Class)
					}
				default:
					t.Errorf("input %q: unexpected base selector type %T", tt.input, base)
				}
			}
		}
	}
}