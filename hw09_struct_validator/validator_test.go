package hw09structvalidator

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type UserRole string

// Test the function on different structures and other types.
type (
	User struct {
		ID     string `json:"id" validate:"len:36"`
		Name   string
		Age    int             `validate:"min:18|max:50"`
		Email  string          `validate:"regexp:^\\w+@\\w+\\.\\w+$"`
		Role   UserRole        `validate:"in:admin,stuff"`
		Phones []string        `validate:"len:11"`
		meta   json.RawMessage //nolint:unused
	}

	App struct {
		Version string `validate:"len:5"`
	}

	Token struct {
		Header    []byte
		Payload   []byte
		Signature []byte
	}

	Response struct {
		Code int    `validate:"in:200,404,500"`
		Body string `json:"omitempty"`
	}
)

// Structures for the nested validation and for the program errors.
type (
	Meta struct {
		Code int    `validate:"in:1,2"`
		Name string `validate:"len:3"`
	}

	Envelope struct {
		Meta  Meta   `validate:"nested"`
		Items []Meta `validate:"nested"`
		Note  string
	}

	Stats struct {
		Values []int `validate:"min:0|max:10"`
	}

	BadLenRule struct {
		Value string `validate:"len:abc"`
	}

	BadRegexpRule struct {
		Value string `validate:"regexp:[a-"`
	}

	BadMinRule struct {
		Value int `validate:"min:ten"`
	}

	BadInRule struct {
		Value int `validate:"in:1,two"`
	}

	UnknownStringRule struct {
		Value string `validate:"length:5"`
	}

	UnknownIntRule struct {
		Value int `validate:"positive:1"`
	}

	UnknownIntRuleWithoutNumber struct {
		Value int `validate:"positive:yes"`
	}

	RuleWithoutValue struct {
		Value string `validate:"len"`
	}

	UnsupportedFieldType struct {
		Value float64 `validate:"min:1"`
	}

	NotAStructNested struct {
		Value int `validate:"nested"`
	}
)

func validUser() User {
	return User{
		ID:     strings.Repeat("a", 36),
		Name:   "Vasya",
		Age:    25,
		Email:  "user@example.com",
		Role:   "admin",
		Phones: []string{"12345678901", "98765432109"},
	}
}

func TestValidateOK(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
	}{
		{name: "valid user", in: validUser()},
		{name: "valid app", in: App{Version: "1.0.0"}},
		{name: "no validate tags at all", in: Token{Header: []byte("h"), Payload: []byte("p")}},
		{name: "valid response", in: Response{Code: 404, Body: "not found"}},
		{name: "empty slice is always valid", in: App{Version: "1.0.0"}},
		{name: "ints in range", in: Stats{Values: []int{0, 5, 10}}},
		{name: "nested structs", in: Envelope{
			Meta:  Meta{Code: 1, Name: "abc"},
			Items: []Meta{{Code: 2, Name: "xyz"}},
			Note:  "ignored",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			t.Parallel()

			require.NoError(t, Validate(tt.in))
		})
	}
}

func TestValidateValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		in       interface{}
		expected ValidationErrors
	}{
		{
			name: "every rule of the user fails",
			in: User{
				ID:     "42",
				Name:   "Vasya",
				Age:    17,
				Email:  "not-an-email",
				Role:   "guest",
				Phones: []string{"12345678901", "123"},
			},
			expected: ValidationErrors{
				{Field: "ID", Err: ErrLen},
				{Field: "Age", Err: ErrMin},
				{Field: "Email", Err: ErrRegexp},
				{Field: "Role", Err: ErrIn},
				{Field: "Phones", Err: ErrLen},
			},
		},
		{
			name:     "age is greater than max",
			in:       func() User { u := validUser(); u.Age = 51; return u }(),
			expected: ValidationErrors{{Field: "Age", Err: ErrMax}},
		},
		{
			name:     "string length",
			in:       App{Version: "1.0"},
			expected: ValidationErrors{{Field: "Version", Err: ErrLen}},
		},
		{
			name:     "int is not in the set",
			in:       Response{Code: 403},
			expected: ValidationErrors{{Field: "Code", Err: ErrIn}},
		},
		{
			name: "every slice element is validated",
			in:   Stats{Values: []int{-1, 5, 42}},
			expected: ValidationErrors{
				{Field: "Values", Err: ErrMin},
				{Field: "Values", Err: ErrMax},
			},
		},
		{
			name: "errors of the nested structs",
			in: Envelope{
				Meta:  Meta{Code: 3, Name: "abc"},
				Items: []Meta{{Code: 1, Name: "ab"}, {Code: 2, Name: "xyz"}},
			},
			expected: ValidationErrors{
				{Field: "Meta.Code", Err: ErrIn},
				{Field: "Items[0].Name", Err: ErrLen},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			t.Parallel()

			err := Validate(tt.in)
			require.Error(t, err)
			require.NotErrorIs(t, err, ErrProgram)
			requireValidationErrors(t, err, tt.expected)
		})
	}
}

func TestValidateProgramErrors(t *testing.T) {
	tests := []struct {
		name     string
		in       interface{}
		expected error
	}{
		{name: "int instead of a struct", in: 42, expected: ErrNotStruct},
		{name: "string instead of a struct", in: "not a struct", expected: ErrNotStruct},
		{name: "nil instead of a struct", in: nil, expected: ErrNotStruct},
		{name: "pointer instead of a struct", in: &App{Version: "1.0.0"}, expected: ErrNotStruct},
		{name: "len value is not a number", in: BadLenRule{Value: "abc"}, expected: ErrInvalidRuleValue},
		{name: "regexp does not compile", in: BadRegexpRule{Value: "abc"}, expected: ErrInvalidRuleValue},
		{name: "min value is not a number", in: BadMinRule{Value: 1}, expected: ErrInvalidRuleValue},
		{name: "in value is not a number", in: BadInRule{Value: 1}, expected: ErrInvalidRuleValue},
		{name: "unknown rule for a string", in: UnknownStringRule{Value: "abc"}, expected: ErrUnknownRule},
		{name: "unknown rule for an int", in: UnknownIntRule{Value: 1}, expected: ErrUnknownRule},
		{
			name:     "unknown rule for an int with a non-numeric value",
			in:       UnknownIntRuleWithoutNumber{Value: 1},
			expected: ErrUnknownRule,
		},
		{name: "rule without a value", in: RuleWithoutValue{Value: "abc"}, expected: ErrInvalidRule},
		{name: "unsupported field type", in: UnsupportedFieldType{Value: 1.5}, expected: ErrUnsupportedType},
		{name: "nested on a non-struct field", in: NotAStructNested{Value: 1}, expected: ErrUnsupportedType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt := tt
			t.Parallel()

			err := Validate(tt.in)
			require.ErrorIs(t, err, ErrProgram)
			require.ErrorIs(t, err, tt.expected)

			var validationErrors ValidationErrors
			require.False(t, errors.As(err, &validationErrors), "program errors must not be validation errors")
		})
	}
}

func TestValidationErrorsError(t *testing.T) {
	err := Validate(App{Version: "1.0"})
	require.Error(t, err)

	var validationErrors ValidationErrors
	require.ErrorAs(t, err, &validationErrors)
	require.Contains(t, err.Error(), "Version")
	require.Contains(t, err.Error(), ErrLen.Error())

	joined := ValidationErrors{
		{Field: "A", Err: ErrLen},
		{Field: "B", Err: ErrMin},
	}
	require.Equal(t, fmt.Sprintf("A: %s; B: %s", ErrLen, ErrMin), joined.Error())
}

func requireValidationErrors(t *testing.T, err error, expected ValidationErrors) {
	t.Helper()

	var actual ValidationErrors
	require.ErrorAs(t, err, &actual)
	require.Len(t, actual, len(expected))

	for i, want := range expected {
		require.Equal(t, want.Field, actual[i].Field)
		require.ErrorIs(t, actual[i].Err, want.Err)
	}
}
