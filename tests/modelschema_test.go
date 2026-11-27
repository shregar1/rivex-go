package rivex_test

import (
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	rivex "github.com/shregar1/rivex-go"
)

type validatorFieldLevel = validator.FieldLevel

type User struct {
	Name     string  `json:"name" validate:"required,min_length=2,max_length=50"`
	Email    string  `json:"email" validate:"required,email"`
	Age      int     `json:"age" validate:"gte=0,lte=150"`
	Username string  `json:"username"`
	Score    float64 `json:"score" validate:"gte=0,lte=100"`
}

func TestModelSchemaBasic(t *testing.T) {
	schema := rivex.NewModelSchema[User]()
	user, err := schema.Validate(map[string]interface{}{
		"name":     "Alice",
		"email":    "alice@example.com",
		"age":      30,
		"username": "alice123",
		"score":    95.5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if user.Name != "Alice" {
		t.Errorf("got name=%s", user.Name)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("got email=%s", user.Email)
	}
	if user.Age != 30 {
		t.Errorf("got age=%d", user.Age)
	}
}

func TestModelSchemaMinLength(t *testing.T) {
	schema := rivex.NewModelSchema[User]()
	_, err := schema.Validate(map[string]interface{}{
		"name":     "A",
		"email":    "alice@example.com",
		"age":      30,
		"username": "alice",
		"score":    50.0,
	})
	if err == nil {
		t.Error("expected error for short name")
	}
}

func TestModelSchemaMaxLength(t *testing.T) {
	schema := rivex.NewModelSchema[User]()
	_, err := schema.Validate(map[string]interface{}{
		"name":     strings.Repeat("a", 100),
		"email":    "alice@example.com",
		"age":      30,
		"username": "alice",
		"score":    50.0,
	})
	if err == nil {
		t.Error("expected error for long name")
	}
}

func TestModelSchemaEmail(t *testing.T) {
	schema := rivex.NewModelSchema[User]()
	_, err := schema.Validate(map[string]interface{}{
		"name":     "Alice",
		"email":    "not-an-email",
		"age":      30,
		"username": "alice",
		"score":    50.0,
	})
	if err == nil {
		t.Error("expected error for invalid email")
	}
}

func TestModelSchemaValueConstraints(t *testing.T) {
	schema := rivex.NewModelSchema[User]()
	_, err := schema.Validate(map[string]interface{}{
		"name":     "Alice",
		"email":    "alice@example.com",
		"age":      200,
		"username": "alice",
		"score":    50.0,
	})
	if err == nil {
		t.Error("expected error for age > 150")
	}
}

func TestModelSchemaTypeCoercion(t *testing.T) {
	schema := rivex.NewModelSchema[User]()
	user, err := schema.Validate(map[string]interface{}{
		"name":     "Alice",
		"email":    "alice@example.com",
		"age":      "30",
		"username": "alice",
		"score":    95,
	})
	if err != nil {
		t.Fatal(err)
	}
	if user.Age != 30 {
		t.Errorf("expected age=30 (coerced from string), got %d", user.Age)
	}
}

func TestModelSchemaExtraFieldsIgnore(t *testing.T) {
	schema := rivex.NewModelSchema[User]()
	_, err := schema.Validate(map[string]interface{}{
		"name":     "Alice",
		"email":    "alice@example.com",
		"age":      30,
		"username": "alice",
		"score":    95.0,
		"unknown":  "extra",
	})
	if err != nil {
		t.Errorf("expected success (extra ignored), got %v", err)
	}
}

func TestModelSchemaExtraFieldsForbid(t *testing.T) {
	schema := rivex.NewModelSchema[User]()
	schema.Extra(rivex.ExtraForbid)
	_, err := schema.Validate(map[string]interface{}{
		"name":     "Alice",
		"email":    "alice@example.com",
		"age":      30,
		"username": "alice",
		"score":    95.0,
		"unknown":  "extra",
	})
	if err == nil {
		t.Error("expected error for extra field")
	}
}

func TestModelSchemaModelDump(t *testing.T) {
	schema := rivex.NewModelSchema[User]()
	user := User{Name: "Alice", Email: "alice@example.com", Age: 30, Username: "alice", Score: 95.0}
	dump, err := schema.ModelDump(user)
	if err != nil {
		t.Fatal(err)
	}
	if dump["name"] != "Alice" {
		t.Errorf("expected name=Alice, got %v", dump["name"])
	}
	if dump["email"] != "alice@example.com" {
		t.Errorf("got email=%v", dump["email"])
	}
}

func TestModelSchemaModelJSON(t *testing.T) {
	schema := rivex.NewModelSchema[User]()
	user := User{Name: "Alice", Email: "alice@example.com", Age: 30, Username: "alice", Score: 95.0}
	data, err := schema.ModelJSON(user)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Alice") {
		t.Error("expected Alice in JSON")
	}
}

func TestModelSchemaModelValidate(t *testing.T) {
	schema := rivex.NewModelSchema[User]()
	rawJSON := []byte(`{"name":"Alice","email":"alice@example.com","age":30,"username":"alice","score":95.0}`)
	user, err := schema.ModelValidate(rawJSON)
	if err != nil {
		t.Fatal(err)
	}
	if user.Name != "Alice" {
		t.Errorf("got name=%s", user.Name)
	}
}

func TestModelSchemaFieldValidator(t *testing.T) {
	schema := rivex.NewModelSchema[User]()
	schema.FieldValidator("username", func(value interface{}, ctx *rivex.ValidationContext) error {
		s, ok := value.(string)
		if !ok {
			return nil
		}
		if !strings.HasPrefix(s, "@") {
			return nil
		}
		return nil
	})
	_, err := schema.Validate(map[string]interface{}{
		"name":     "Alice",
		"email":    "alice@example.com",
		"age":      30,
		"username": "@alice",
		"score":    95.0,
	})
	if err != nil {
		t.Errorf("expected success, got %v", err)
	}
}

func TestModelSchemaModelValidator(t *testing.T) {
	schema := rivex.NewModelSchema[User]()
	schema.ModelValidator(func(values map[string]interface{}, ctx *rivex.ValidationContext) error {
		if _, ok := values["age"]; ok {
			_ = ageValidator
		}
		return nil
	})
	_, err := schema.Validate(map[string]interface{}{
		"name":     "Alice",
		"email":    "alice@example.com",
		"age":      30,
		"username": "alice",
		"score":    95.0,
	})
	if err != nil {
		t.Errorf("expected success for age=30, got %v", err)
	}
}

func TestModelSchemaFieldSerializer(t *testing.T) {
	schema := rivex.NewModelSchema[User]()
	schema.FieldSerializer("email", func(value interface{}) interface{} {
		return "***REDACTED***"
	})
	user := User{Name: "Alice", Email: "secret@example.com"}
	dump, _ := schema.ModelDump(user)
	if dump["email"] != "***REDACTED***" {
		t.Errorf("expected redacted email, got %v", dump["email"])
	}
}

func TestModelSchemaComputedField(t *testing.T) {
	type UserWithDisplay struct {
		Name  string `json:"name" validate:"required"`
		Email string `json:"email" validate:"required,email"`
	}
	schema := rivex.NewModelSchema[UserWithDisplay]()
	schema.ComputedField("DisplayName", func(values map[string]interface{}) interface{} {
		name, _ := values["name"].(string)
		return "User: " + name
	})
	rawJSON := []byte(`{"name":"Alice","email":"alice@example.com"}`)
	_, err := schema.ModelValidate(rawJSON)
	if err != nil {
		t.Fatal(err)
	}
}

func TestModelSchemaBuilder(t *testing.T) {
	b := rivex.NewModelSchemaBuilder[User]()
	b.Extra(rivex.ExtraIgnore)
	schema := b.Build()
	_, err := schema.Validate(map[string]interface{}{
		"name":     "Alice",
		"email":    "alice@example.com",
		"age":      30,
		"username": "alice",
		"score":    95.0,
	})
	if err != nil {
		t.Errorf("expected success, got %v", err)
	}
}

func TestModelSchemaFields(t *testing.T) {
	schema := rivex.NewModelSchema[User]()
	fields := schema.Fields()
	if len(fields) != 5 {
		t.Errorf("expected 5 fields, got %d", len(fields))
	}
}

func TestModelSchemaFrozen(t *testing.T) {
	schema := rivex.NewModelSchema[User]()
	schema.Frozen(true)
	if !schema.Config().Frozen {
		t.Error("expected frozen")
	}
}

func TestModelSchemaValidationReport(t *testing.T) {
	schema := rivex.NewModelSchema[User]()
	_, report := schema.ValidateWithReport(map[string]interface{}{
		"name":     "A",
		"email":    "alice@example.com",
		"age":      30,
		"username": "alice",
		"score":    95.0,
	})
	if report == nil {
		t.Fatal("expected report")
	}
	if report.TotalErrors == 0 {
		t.Error("expected errors")
	}
}

func TestDiscriminatedUnion(t *testing.T) {
	du := rivex.NewDiscriminatedUnion("type")
	type Cat struct{ Name string }
	type Dog struct{ Breed string }
	du.Register("cat", func(data map[string]interface{}) (interface{}, error) {
		name, _ := data["name"].(string)
		return Cat{Name: name}, nil
	})
	du.Register("dog", func(data map[string]interface{}) (interface{}, error) {
		breed, _ := data["breed"].(string)
		return Dog{Breed: breed}, nil
	})
	result, err := du.Resolve(map[string]interface{}{
		"type": "cat",
		"name": "Whiskers",
	})
	if err != nil {
		t.Fatal(err)
	}
	cat, ok := result.(Cat)
	if !ok {
		t.Fatalf("expected Cat, got %T", result)
	}
	if cat.Name != "Whiskers" {
		t.Errorf("got name=%s", cat.Name)
	}
}

func TestDiscriminatedUnionUnknownTag(t *testing.T) {
	du := rivex.NewDiscriminatedUnion("type")
	_, err := du.Resolve(map[string]interface{}{"type": "unknown"})
	if err == nil {
		t.Error("expected error for unknown tag")
	}
}

func TestNestedSchemaValidator(t *testing.T) {
	type Address struct {
		Street string `json:"street" validate:"required,min_length=1"`
		City   string `json:"city"`
	}
	nested := rivex.NewNestedSchemaValidator[Address]()
	address, err := nested.Validate(map[string]interface{}{
		"street": "123 Main St",
		"city":   "Springfield",
	})
	if err != nil {
		t.Fatal(err)
	}
	if address.Street != "123 Main St" {
		t.Errorf("got street=%s", address.Street)
	}
}

func TestFieldMeta(t *testing.T) {
	schema := rivex.NewModelSchema[User]()
	meta := schema.ModelField("Name")
	if meta == nil {
		t.Fatal("expected meta")
	}
	if meta.Name != "name" {
		t.Errorf("got %s", meta.Name)
	}
}

func TestAddFieldProgrammatic(t *testing.T) {
	type Dynamic struct {
		ID int `json:"id" validate:"required"`
	}
	schema := rivex.NewModelSchema[Dynamic]()
	schema.AddField("CustomField", "", rivex.WithModelFieldDescription("custom field"))
	_, err := schema.Validate(map[string]interface{}{
		"id":          42,
		"customfield": "hello",
	})
	if err != nil {
		t.Errorf("expected success, got %v", err)
	}
}

func TestModelSchemaCustomEngineValidator(t *testing.T) {
	type Post struct {
		Title  string `json:"title" validate:"required"`
		Slug   string `json:"slug" validate:"required,slug"`
		Likes  int    `json:"likes" validate:"gte=0"`
	}
	schema := rivex.NewModelSchema[Post]()
	err := schema.RegisterValidation("slug", func(fl validatorFieldLevel) bool {
		return schema.Engine().Var(fl.Field().String(), "alphanum") == nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = schema.Validate(map[string]interface{}{
		"title": "Hello",
		"slug":  "hello-world",
		"likes": 0,
	})
	if err == nil {
		t.Error("expected slug validation error")
	}
}

var ageValidator = 0

func TestValidationContextErrorMap(t *testing.T) {
	schema := rivex.NewModelSchema[User]()
	_, err := schema.Validate(map[string]interface{}{
		"name":     "A",
		"email":    "alice@example.com",
		"age":      30,
		"username": "alice",
		"score":    95.0,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var verrs *rivex.ValidationErrors
	if !errorsAs(err, &verrs) {
		t.Fatal("expected ValidationErrors type")
	}
	em := verrs.Context.ErrorMap()
	if len(em) == 0 {
		t.Error("expected errors in map")
	}
}

func errorsAs(err error, target interface{}) bool {
	type asInterface interface{ As(interface{}) bool }
	if a, ok := err.(asInterface); ok {
		return a.As(target)
	}
	return false
}
