package hw09structvalidator

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

var ErrProgram = errors.New("validator")

var (
	ErrNotStruct        = fmt.Errorf("%w: only a struct can be validated", ErrProgram)
	ErrInvalidRule      = fmt.Errorf(`%w: rule must look like "name:value"`, ErrProgram)
	ErrUnknownRule      = fmt.Errorf("%w: unknown rule", ErrProgram)
	ErrInvalidRuleValue = fmt.Errorf("%w: invalid rule value", ErrProgram)
	ErrUnsupportedType  = fmt.Errorf("%w: unsupported field type", ErrProgram)
)

var (
	ErrLen    = errors.New("unexpected string length")
	ErrRegexp = errors.New("string does not match regexp")
	ErrIn     = errors.New("value is not in the allowed set")
	ErrMin    = errors.New("value is less than the minimum")
	ErrMax    = errors.New("value is greater than the maximum")
)

const tagName = "validate"
const nestedRule = "nested"

type ValidationError struct {
	Field string
	Err   error
}

func (v ValidationError) Error() string {
	return v.Field + ": " + v.Err.Error()
}

func (v ValidationError) Unwrap() error {
	return v.Err
}

type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	var sb strings.Builder
	for i, e := range v {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(e.Error())
	}
	return sb.String()
}

func Validate(v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("%w, got %T", ErrNotStruct, v)
	}

	errs, err := validateStruct(rv, "")
	if err != nil {
		return err
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func validateStruct(rv reflect.Value, prefix string) (ValidationErrors, error) {
	rt := rv.Type()
	errs := make(ValidationErrors, 0, rt.NumField())

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if !field.IsExported() {
			continue
		}

		tag, ok := field.Tag.Lookup(tagName)
		if !ok || tag == "" {
			continue
		}

		fieldErrs, err := validateField(prefix+field.Name, rv.Field(i), tag)
		if err != nil {
			return nil, err
		}
		errs = append(errs, fieldErrs...)
	}

	return errs, nil
}

func validateField(name string, fv reflect.Value, tag string) (ValidationErrors, error) {
	if tag == nestedRule {
		return validateNested(name, fv)
	}

	rules := strings.Split(tag, "|")
	if fv.Kind() != reflect.Slice {
		return validateValue(name, fv, rules)
	}

	errs := make(ValidationErrors, 0, fv.Len())
	for i := 0; i < fv.Len(); i++ {
		itemErrs, err := validateValue(name, fv.Index(i), rules)
		if err != nil {
			return nil, err
		}
		errs = append(errs, itemErrs...)
	}
	return errs, nil
}

func validateNested(name string, fv reflect.Value) (ValidationErrors, error) {
	if fv.Kind() == reflect.Struct {
		return validateStruct(fv, name+".")
	}

	if fv.Kind() != reflect.Slice || fv.Type().Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: %s is %s, not a struct", ErrUnsupportedType, name, fv.Kind())
	}

	errs := make(ValidationErrors, 0, fv.Len())
	for i := 0; i < fv.Len(); i++ {
		itemErrs, err := validateStruct(fv.Index(i), fmt.Sprintf("%s[%d].", name, i))
		if err != nil {
			return nil, err
		}
		errs = append(errs, itemErrs...)
	}
	return errs, nil
}

func validateValue(name string, v reflect.Value, rules []string) (ValidationErrors, error) {
	errs := make(ValidationErrors, 0, len(rules))

	for _, rule := range rules {
		err := applyRule(v, rule)
		switch {
		case err == nil:
		case errors.Is(err, ErrProgram):
			return nil, fmt.Errorf("field %s: %w", name, err)
		default:
			errs = append(errs, ValidationError{Field: name, Err: err})
		}
	}

	return errs, nil
}

func applyRule(v reflect.Value, rule string) error {
	name, arg, ok := strings.Cut(rule, ":")
	if !ok {
		return fmt.Errorf("%w, got %q", ErrInvalidRule, rule)
	}

	switch {
	case v.Kind() == reflect.String:
		return applyStringRule(v.String(), name, arg)
	case v.CanInt():
		return applyIntRule(v.Int(), name, arg)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedType, v.Kind())
	}
}

func applyStringRule(s, rule, arg string) error {
	switch rule {
	case "len":
		length, err := strconv.Atoi(arg)
		if err != nil {
			return fmt.Errorf("%w: len:%s", ErrInvalidRuleValue, arg)
		}
		if actual := utf8.RuneCountInString(s); actual != length {
			return fmt.Errorf("%w: expected %d, got %d", ErrLen, length, actual)
		}
	case "regexp":
		re, err := regexp.Compile(arg)
		if err != nil {
			return fmt.Errorf("%w: regexp:%s: %w", ErrInvalidRuleValue, arg, err)
		}
		if !re.MatchString(s) {
			return fmt.Errorf("%w: %q does not match %q", ErrRegexp, s, arg)
		}
	case "in":
		set := strings.Split(arg, ",")
		if !slices.Contains(set, s) {
			return fmt.Errorf("%w: %q is not in {%s}", ErrIn, s, arg)
		}
	default:
		return fmt.Errorf("%w %q for a string", ErrUnknownRule, rule)
	}
	return nil
}

func applyIntRule(n int64, rule, arg string) error {
	switch rule {
	case "in":
		return applyIntInRule(n, arg)
	case "min", "max":
		return applyIntLimitRule(n, rule, arg)
	default:
		return fmt.Errorf("%w %q for an integer", ErrUnknownRule, rule)
	}
}

func applyIntLimitRule(n int64, rule, arg string) error {
	limit, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: %s:%s", ErrInvalidRuleValue, rule, arg)
	}

	if rule == "min" && n < limit {
		return fmt.Errorf("%w: %d is less than %d", ErrMin, n, limit)
	}
	if rule == "max" && n > limit {
		return fmt.Errorf("%w: %d is greater than %d", ErrMax, n, limit)
	}
	return nil
}

func applyIntInRule(n int64, arg string) error {
	found := false
	for _, item := range strings.Split(arg, ",") {
		allowed, err := strconv.ParseInt(item, 10, 64)
		if err != nil {
			return fmt.Errorf("%w: in:%s", ErrInvalidRuleValue, arg)
		}
		if n == allowed {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("%w: %d is not in {%s}", ErrIn, n, arg)
	}
	return nil
}
