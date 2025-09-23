package main

import (
	"encoding"
	"fmt"
	"reflect"
	"strings"

	"github.com/alecthomas/kong"
)

// toStrings creates a []string from any number of stringers.
// It is useful for succinctly creating slices that can be passed to strings.Join().
func toStrings(in ...fmt.Stringer) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = v.String()
	}
	return out
}

// joinStringers is a convenience wrapper for using toStrings() with strings.Join().
func joinStringers(sep string, elems ...fmt.Stringer) string {
	return strings.Join(toStrings(elems...), sep)
}

func joinQuoted(sep string, elems ...any) string {
	ss := make([]string, len(elems))
	for i, elem := range elems {
		ss[i] = fmt.Sprintf("%q", elem)
	}
	return strings.Join(ss, sep)
}

// enumTag comma-joins stringer values as Kong expects for struct field `enum:` tags.
func enumTag(elems ...fmt.Stringer) string {
	return joinStringers(",", elems...)
}

// RegisterEnumPlaceholderMapperOpt is a shortcut for creating a kong.Option
// that registers a mapper for type T.
func RegisterEnumPlaceholderMapperOpt[T any]() kong.Option {
	var val T
	return kong.TypeMapper(reflect.TypeOf(val), enumPlaceholderMapper[T]{})
}

// enumPlaceholderMapper is a mapper that can be registered in Kong's mapper registry
// to generate placeholders in help text that follow conventional (manpage, GNU, etc.)
// formatting for CLI flags with enum choices.
// It should not be used for fields that do not have an `enum:` tag.
type enumPlaceholderMapper[T any] struct{}

// Decode is required in order to register the PlaceHolderProvider interface,
// so we reimplement kong.textUnmarshalerAdapter.Decode (which is not exported)
// to keep things minimal. The only difference is that we fall back to Kong's generic
// decoding behavior in edge-cases where a type registered for this decoder doesn't
// actually implement encoding.TextUnmarshaler.
func (enumPlaceholderMapper[T]) Decode(ctx *kong.DecodeContext, target reflect.Value) error {
	// Read raw token as a string
	var value string
	if err := ctx.Scan.PopValueInto("value", &value); err != nil {
		return err
	}

	// If the destination implements TextUnmarshaler, use it directly.
	if tu, ok := target.Addr().Interface().(encoding.TextUnmarshaler); ok {
		return tu.UnmarshalText([]byte(value))
	}

	// Allow fallback to more generic decoding (shouldn't happen if we're only
	// registering this mapper for types that implement encoding.TextUnmarshaler).
	// NOTE: Kong's error message isn't as user-friendly here, so it should be avoided.
	return ctx.Scan.PopValueInto("value", target.Addr().Interface())
}

// PlaceHolder creates manpage-style placeholders for enum CLI flags.
// If the flag already has a placeholder (e.g. from its field tag), it is unaltered.
func (enumPlaceholderMapper[T]) PlaceHolder(flag *kong.Flag) string {
	if flag.PlaceHolder != "" {
		// Don't change explicitly-set `placeholder:` tag values
		return flag.PlaceHolder
	}
	if flag.Enum == "" {
		// Flag does not have an `enum:` tag we can use to generate a placeholder
		return ""
	}
	// Convert "a,b,c" -> "{a|b|c}"
	return "{" + strings.ReplaceAll(flag.Enum, ",", "|") + "}"
}
