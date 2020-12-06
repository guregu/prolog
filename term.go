package prolog

import (
	"fmt"
	"strconv"
	"strings"
)

type Term interface {
	fmt.Stringer
	TermString(operators) string
	Unify(Term) bool
	Simplify() Term
}

type Atom string

func (a Atom) String() string {
	return a.TermString(nil)
}

func (a Atom) TermString(operators) string {
	return string(a)
}

func (a Atom) Unify(t Term) bool {
	switch t := t.(type) {
	case Atom:
		return a == t
	case *Variable:
		return t.Unify(a)
	default:
		return false
	}
}

func (a Atom) Simplify() Term {
	return a
}

type Integer int64

func (i Integer) String() string {
	return i.TermString(nil)
}

func (i Integer) TermString(operators) string {
	return strconv.FormatInt(int64(i), 10)
}

func (i Integer) Unify(t Term) bool {
	switch t := t.(type) {
	case Integer:
		return i == t
	case *Variable:
		return t.Unify(i)
	default:
		return false
	}
}

func (i Integer) Simplify() Term {
	return i
}

type Variable struct {
	Name string
	Ref  Term
}

func (v *Variable) String() string {
	return v.TermString(nil)
}

func (v *Variable) TermString(os operators) string {
	name := v.Name
	if name == "" {
		name = fmt.Sprintf("_%p", v)
	}
	if v.Ref == nil {
		return name
	}
	return fmt.Sprintf("%s = %s", name, v.Ref.TermString(os))
}

func (v *Variable) Unify(t Term) bool {
	if v.Ref != nil {
		return v.Ref.Unify(t)
	}
	v.Ref = t
	return true
}

func (v *Variable) Simplify() Term {
	if v.Ref == nil {
		return v
	}
	return v.Ref.Simplify()
}

type Compound struct {
	Functor Atom
	Args    []Term
}

func (c *Compound) String() string {
	return c.TermString(nil)
}

func (c *Compound) TermString(os operators) string {
	if c.Functor == "." && len(c.Args) == 2 { // list
		t := Term(c)
		var (
			elems []string
			rest  string
		)
		for {
			if l, ok := t.(*Compound); ok && l.Functor == "." && len(l.Args) == 2 {
				elems = append(elems, l.Args[0].TermString(os))
				t = l.Args[1]
				continue
			}
			if a, ok := t.(Atom); ok && a == "[]" {
				break
			}
			rest = "|" + t.TermString(os)
			break
		}
		return fmt.Sprintf("[%s%s]", strings.Join(elems, ", "), rest)
	}

	switch len(c.Args) {
	case 1:
		for _, o := range os {
			if o.Name != c.Functor {
				continue
			}
			switch o.Type {
			case `xf`, `yf`:
				return fmt.Sprintf("%s%s", c.Args[0].TermString(os), c.Functor.TermString(os))
			case `fx`, `fy`:
				return fmt.Sprintf("%s%s", c.Functor.TermString(os), c.Args[0].TermString(os))
			}
		}
	case 2:
		for _, o := range os {
			if o.Name != c.Functor {
				continue
			}
			switch o.Type {
			case `xfx`, `xfy`, `yfx`:
				return fmt.Sprintf("%s%s%s", c.Args[0].TermString(os), c.Functor.TermString(os), c.Args[1].TermString(os))
			}
		}
	}

	args := make([]string, len(c.Args))
	for i, arg := range c.Args {
		args[i] = arg.TermString(os)
	}
	return fmt.Sprintf("%s(%s)", c.Functor.TermString(os), strings.Join(args, ", "))
}

func (c *Compound) Unify(t Term) bool {
	switch t := t.(type) {
	case *Compound:
		if c.Functor != t.Functor {
			return false
		}
		if len(c.Args) != len(t.Args) {
			return false
		}
		for i := range c.Args {
			if !c.Args[i].Unify(t.Args[i]) {
				return false
			}
		}
		return true
	case *Variable:
		return t.Unify(c)
	default:
		return false
	}
}

func (c *Compound) Simplify() Term {
	args := make([]Term, len(c.Args))
	for i := range args {
		args[i] = c.Args[i].Simplify()
	}
	return &Compound{Functor: c.Functor, Args: args}
}

func Cons(car, cdr Term) Term {
	return &Compound{
		Functor: ".",
		Args:    []Term{car, cdr},
	}
}

func List(ts ...Term) Term {
	l := Term(Atom("[]"))
	for i := len(ts) - 1; i >= 0; i-- {
		l = Cons(ts[i], l)
	}
	return l
}

func Resolve(t Term) Term {
	for t != nil {
		switch v := t.(type) {
		case Atom, Integer, *Compound:
			return v
		case *Variable:
			t = v.Ref
		}
	}
	return nil
}