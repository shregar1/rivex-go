package rivex_test

import (
	"strings"
	"testing"

	rivex "github.com/shregar1/rivex-go"
)

func TestSchemaBasicField(t *testing.T) {
	s := rivex.NewSchema()
	s.Field("name", rivex.FieldString, rivex.Required())
	s.Field("age", rivex.FieldInt)

	data := map[string]interface{}{
		"name": "Alice",
		"age":  30,
	}
	result := s.Validate(data)
	if result["name"] != "Alice" {
		t.Errorf("expected name=Alice, got %v", result["name"])
	}
	if result["age"] != 30 {
		t.Errorf("expected age=30, got %v", result["age"])
	}
}

func TestSchemaRequiredMissing(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for missing required field")
		}
	}()
	s := rivex.NewSchema()
	s.Field("name", rivex.FieldString, rivex.Required())
	s.Validate(map[string]interface{}{})
}

func TestSchemaDefaultValue(t *testing.T) {
	s := rivex.NewSchema()
	s.Field("count", rivex.FieldInt, rivex.Default(10))
	result := s.Validate(map[string]interface{}{})
	if result["count"] != 10 {
		t.Errorf("expected default count=10, got %v", result["count"])
	}
}

func TestSchemaStringCoercion(t *testing.T) {
	s := rivex.NewSchema()
	s.Field("count", rivex.FieldInt)
	result := s.Validate(map[string]interface{}{"count": "42"})
	if result["count"] != 42 {
		t.Errorf("expected 42, got %v", result["count"])
	}
}

func TestSchemaBoolCoercion(t *testing.T) {
	s := rivex.NewSchema()
	s.Field("active", rivex.FieldBool)
	result := s.Validate(map[string]interface{}{"active": "true"})
	if result["active"] != true {
		t.Errorf("expected true, got %v", result["active"])
	}
}

func TestSchemaEmailValidator(t *testing.T) {
	s := rivex.NewSchema()
	s.Field("email", rivex.FieldString, rivex.Validator(rivex.Email()))

	data := map[string]interface{}{"email": "user@example.com"}
	result := s.Validate(data)
	if result["email"] != "user@example.com" {
		t.Errorf("got %v", result["email"])
	}
}

func TestSchemaMinLengthValidator(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for too short string")
		}
	}()
	s := rivex.NewSchema()
	s.Field("password", rivex.FieldString, rivex.Required(), rivex.Validator(rivex.MinLength(8)))
	s.Validate(map[string]interface{}{"password": "short"})
}

func TestSchemaInValuesValidator(t *testing.T) {
	s := rivex.NewSchema()
	s.Field("role", rivex.FieldString, rivex.Validator(rivex.InValues("admin", "user", "guest")))
	result := s.Validate(map[string]interface{}{"role": "admin"})
	if result["role"] != "admin" {
		t.Errorf("got %v", result["role"])
	}
}

func TestSchemaAliasLookup(t *testing.T) {
	s := rivex.NewSchema()
	s.Field("name", rivex.FieldString, rivex.Aliases("fullName", "displayName"))
	result := s.Validate(map[string]interface{}{"fullName": "Alice"})
	if result["name"] != "Alice" {
		t.Errorf("expected alias to resolve, got %v", result["name"])
	}
}

func TestSchemaToJSON(t *testing.T) {
	s := rivex.NewSchema()
	s.Field("name", rivex.FieldString)
	data, err := s.ToJSON(map[string]interface{}{"name": "Alice", "extra": "data"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Alice") {
		t.Errorf("expected JSON to contain Alice: %s", string(data))
	}
}

func TestFieldValidators(t *testing.T) {
	tests := []struct {
		name      string
		validator rivex.FieldValidator
		value     interface{}
		shouldErr bool
	}{
		{"min-length-ok", rivex.MinLength(3), "abc", false},
		{"min-length-fail", rivex.MinLength(3), "ab", true},
		{"max-length-ok", rivex.MaxLength(5), "hello", false},
		{"max-length-fail", rivex.MaxLength(5), "hello world", true},
		{"min-ok", rivex.Min(10), 15, false},
		{"min-fail", rivex.Min(10), 5, true},
		{"max-ok", rivex.Max(10), 5, false},
		{"max-fail", rivex.Max(10), 15, true},
		{"email-ok", rivex.Email(), "user@example.com", false},
		{"email-fail", rivex.Email(), "not-an-email", true},
		{"url-ok", rivex.URL(), "https://example.com", false},
		{"url-fail", rivex.URL(), "not-a-url", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator(tt.value)
			if tt.shouldErr && err == nil {
				t.Error("expected error")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestModelSerializer(t *testing.T) {
	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Age   int    `json:"age"`
	}
	user := User{Name: "Alice", Email: "alice@example.com", Age: 30}
	ms := rivex.NewModelSerializer(user)
	if ms.Schema == nil {
		t.Fatal("expected schema to be built")
	}
	fields := ms.Schema.Fields()
	if _, ok := fields["name"]; !ok {
		t.Error("expected name field")
	}
	if _, ok := fields["email"]; !ok {
		t.Error("expected email field")
	}
	if _, ok := fields["age"]; !ok {
		t.Error("expected age field")
	}
}

func TestListSerializer(t *testing.T) {
	itemSerializer := rivex.WrapSerializer(func(v interface{}) (interface{}, error) {
		return v, nil
	})
	listSerializer := rivex.NewListSerializer(itemSerializer)
	items := []map[string]interface{}{
		{"a": 1},
		{"a": 2},
	}
	result, err := listSerializer.Serialize(items)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
}

func TestSerializerRegistry(t *testing.T) {
	reg := rivex.NewSerializerRegistry()
	reg.Register("custom", rivex.WrapSerializer(func(v interface{}) (interface{}, error) {
		return "wrapped", nil
	}))
	s, ok := reg.Get("custom")
	if !ok {
		t.Fatal("expected custom serializer")
	}
	result, err := s.Serialize("input")
	if err != nil {
		t.Fatal(err)
	}
	if result != "wrapped" {
		t.Errorf("expected wrapped, got %v", result)
	}
	if len(reg.Names()) != 1 {
		t.Errorf("expected 1 serializer, got %d", len(reg.Names()))
	}
}

func TestWrapSerializer(t *testing.T) {
	s := rivex.WrapSerializer(func(v interface{}) (interface{}, error) {
		if v == "x" {
			return "y", nil
		}
		return v, nil
	})
	result, err := s.Serialize("x")
	if err != nil {
		t.Fatal(err)
	}
	if result != "y" {
		t.Errorf("expected y, got %v", result)
	}
}
