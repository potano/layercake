package fns

import "testing"


func TestTemplate(t *testing.T) {
	expanded := Template("some text", map[string]string{})
	if expanded != "some text" {
		t.Fatalf("no-substitution template returned '%s'", expanded)
	}
	expanded = Template("text with {sub} inserted", map[string]string{"sub": "something"})
	if expanded != "text with something inserted" {
		t.Errorf("simple substitution returned '%s'", expanded)
	}
	expanded = Template("{sub} at beginning", map[string]string{"sub": "substitution"})
	if expanded != "substitution at beginning" {
		t.Errorf("item at start returned '%s'", expanded)
	}
	expanded = Template("a {1}{2} punch", map[string]string{"1": "one", "2": "-two"})
	if expanded != "a one-two punch" {
		t.Errorf("adjacent substitutions returned '%s'", expanded)
	}
	expanded = Template("substitution at {place}", map[string]string{"place": "end"})
	if expanded != "substitution at end" {
		t.Errorf("item at end returned '%s'", expanded)
	}
	expanded = Template("{missing} in action", map[string]string{"not missing": "xx"})
	if expanded != " in action" {
		t.Errorf("missing item at start returned '%s'", expanded)
	}
	expanded = Template("one {here}, one {not} there", map[string]string{"here": "here"})
	if expanded != "one here, one  there" {
		t.Errorf("non-adjacent items, one present, one not, returned '%s'", expanded)
	}
	expanded = Template("{in}{flammable}", map[string]string{"flammable": "flammable"})
	if expanded != "flammable" {
		t.Errorf("adjacent items, one present, one not, returned '%s'", expanded)
	}
	expanded = Template("{a}{b} {a}{b}", map[string]string{"a": "tip", "b": "-top"})
	if expanded != "tip-top tip-top" {
		t.Errorf("multiple insertions of items returned '%s'", expanded)
	}
	expanded = Template("subst} not opened properly", map[string]string{"subst": "key"})
	if expanded != "subst} not opened properly" {
		t.Errorf("item w/o opening { returned '%s'", expanded)
	}
	expanded = Template("subst} not {o} properly", map[string]string{"o": "opened"})
	if expanded != "subst} not opened properly" {
		t.Errorf("item w/ opening { plus correctly-formed item returned '%s'", expanded)
	}
	expanded = Template("template not {ended", map[string]string{"ended": "ended"})
	if expanded != "template not " {
		t.Errorf("template w/o ending } returned '%s'", expanded)
	}
	expanded = Template("{} ex nihilo", map[string]string{"": "creatio"})
	if expanded != "creatio ex nihilo" {
		t.Errorf("item w/ zero-length key returned '%s'", expanded)
	}
	expanded = Template(`should \{see} marker`, map[string]string{"see": "not see"})
	if expanded != "should {see} marker" {
		t.Errorf("escaped item returned '%s'", expanded)
	}
	expanded = Template(`should \\{see} marker`, map[string]string{"see": "see"})
	if expanded != "should \\see marker" {
		t.Errorf("escaped backslash before marker returned '%s'", expanded)
	}
}

