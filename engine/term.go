package engine

import (
	"fmt"
	"io"
	"strings"
)

// Term is a prolog term.
type Term interface {
	fmt.Stringer
	Unify(Term, bool, *Env) (*Env, bool)
	Unparse(func(Token), WriteTermOptions, *Env)
}

// Contains checks if t contains s.
func Contains(t, s Term, env *Env) bool {
	switch t := t.(type) {
	case Variable:
		if t == s {
			return true
		}
		ref, ok := env.Lookup(t)
		if !ok {
			return false
		}
		return Contains(ref, s, env)
	case *Compound:
		if s, ok := s.(Atom); ok && t.Functor == s {
			return true
		}
		for _, a := range t.Args {
			if Contains(a, s, env) {
				return true
			}
		}
		return false
	default:
		return t == s
	}
}

// Rulify returns t if t is in a form of P:-Q, t:-true otherwise.
func Rulify(t Term, env *Env) Term {
	t = env.Resolve(t)
	if c, ok := t.(*Compound); ok && c.Functor == ":-" && len(c.Args) == 2 {
		return t
	}
	return &Compound{Functor: ":-", Args: []Term{t, Atom("true")}}
}

// WriteTermOptions describes options to write terms.
type WriteTermOptions struct {
	Quoted     bool
	Ops        Operators
	NumberVars bool

	Priority int
}

func (o WriteTermOptions) withPriority(p int) WriteTermOptions {
	ret := o
	ret.Priority = p
	return ret
}

var defaultWriteTermOptions = WriteTermOptions{
	Quoted: true,
	Ops: Operators{
		{Priority: 500, Specifier: OperatorSpecifierYFX, Name: "+"}, // for flag+value
		{Priority: 400, Specifier: OperatorSpecifierYFX, Name: "/"}, // for principal functors
	},
	Priority: 1200,
}

func compare(a, b Term, env *Env) int64 {
	switch a := env.Resolve(a).(type) {
	case Variable:
		switch b := env.Resolve(b).(type) {
		case Variable:
			return int64(strings.Compare(string(a), string(b)))
		default:
			return -1
		}
	case Float:
		switch b := env.Resolve(b).(type) {
		case Variable:
			return 1
		case Float:
			return int64(a - b)
		case Integer:
			if d := int64(a - Float(b)); d != 0 {
				return d
			}
			return -1
		default:
			return -1
		}
	case Integer:
		switch b := env.Resolve(b).(type) {
		case Variable:
			return 1
		case Float:
			d := int64(Float(a) - b)
			if d == 0 {
				return 1
			}
			return d
		case Integer:
			return int64(a - b)
		default:
			return -1
		}
	case Atom:
		switch b := env.Resolve(b).(type) {
		case Variable, Float, Integer:
			return 1
		case Atom:
			return int64(strings.Compare(string(a), string(b)))
		default:
			return -1
		}
	case *Compound:
		switch b := b.(type) {
		case *Compound:
			if d := compare(a.Functor, b.Functor, env); d != 0 {
				return d
			}

			if d := len(a.Args) - len(b.Args); d != 0 {
				return int64(d)
			}

			for i := range a.Args {
				if d := compare(a.Args[i], b.Args[i], env); d != 0 {
					return d
				}
			}

			return 0
		default:
			return 1
		}
	default:
		return 1
	}
}

// Write outputs one of the external representations of the term.
func Write(w io.Writer, t Term, opts WriteTermOptions, env *Env) error {
	var (
		last TokenKind
		err  error
	)
	env.Resolve(t).Unparse(func(token Token) {
		if err != nil {
			return
		}
		var sb strings.Builder
		if spacing[last][token.Kind] {
			_, _ = sb.WriteString(" ")
		}
		_, _ = sb.WriteString(token.Val)
		last = token.Kind
		_, err = fmt.Fprint(w, sb.String())
	}, opts, env)
	return err
}