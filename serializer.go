package rivex

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Serializer interface {
	Serialize(interface{}) (interface{}, error)
	Deserialize(interface{}, interface{}) error
}

type Schema struct {
	fields map[string]*Field
	mu     sync.RWMutex
}

type Field struct {
	Name        string
	Type        FieldType
	Required    bool
	Default     interface{}
	Aliases     []string
	Validators  []FieldValidator
	Description string
	Example     interface{}
	Deprecated  bool
}

type FieldType int

const (
	FieldString  FieldType = iota
	FieldInt
	FieldFloat
	FieldBool
	FieldTime
	FieldJSON
	FieldSlice
	FieldMap
	FieldAny
)

type FieldValidator func(value interface{}) error

func NewSchema() *Schema {
	return &Schema{
		fields: make(map[string]*Field),
	}
}

func (s *Schema) Field(name string, fieldType FieldType, opts ...FieldOption) *Field {
	f := &Field{
		Name: name,
		Type: fieldType,
	}
	for _, opt := range opts {
		opt(f)
	}
	s.mu.Lock()
	s.fields[name] = f
	s.mu.Unlock()
	return f
}

type FieldOption func(*Field)

func Required() FieldOption {
	return func(f *Field) {
		f.Required = true
	}
}

func Default(value interface{}) FieldOption {
	return func(f *Field) {
		f.Default = value
	}
}

func Aliases(aliases ...string) FieldOption {
	return func(f *Field) {
		f.Aliases = aliases
	}
}

func Description(desc string) FieldOption {
	return func(f *Field) {
		f.Description = desc
	}
}

func Example(example interface{}) FieldOption {
	return func(f *Field) {
		f.Example = example
	}
}

func Deprecated() FieldOption {
	return func(f *Field) {
		f.Deprecated = true
	}
}

func Validator(fn FieldValidator) FieldOption {
	return func(f *Field) {
		f.Validators = append(f.Validators, fn)
	}
}

func MinLength(min int) FieldValidator {
	return func(value interface{}) error {
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string")
		}
		if len(s) < min {
			return fmt.Errorf("minimum length is %d", min)
		}
		return nil
	}
}

func MaxLength(max int) FieldValidator {
	return func(value interface{}) error {
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string")
		}
		if len(s) > max {
			return fmt.Errorf("maximum length is %d", max)
		}
		return nil
	}
}

func Min(minVal float64) FieldValidator {
	return func(value interface{}) error {
		switch v := value.(type) {
		case int:
			if float64(v) < minVal {
				return fmt.Errorf("minimum value is %v", minVal)
			}
		case float64:
			if v < minVal {
				return fmt.Errorf("minimum value is %v", minVal)
			}
		}
		return nil
	}
}

func Max(maxVal float64) FieldValidator {
	return func(value interface{}) error {
		switch v := value.(type) {
		case int:
			if float64(v) > maxVal {
				return fmt.Errorf("maximum value is %v", maxVal)
			}
		case float64:
			if v > maxVal {
				return fmt.Errorf("maximum value is %v", maxVal)
			}
		}
		return nil
	}
}

func Pattern(regex string) FieldValidator {
	return func(value interface{}) error {
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string")
		}
		if !regexMatch(regex, s) {
			return fmt.Errorf("does not match pattern %s", regex)
		}
		return nil
	}
}

func Email() FieldValidator {
	return func(value interface{}) error {
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string")
		}
		if !ValidateEmail(s) {
			return fmt.Errorf("invalid email address")
		}
		return nil
	}
}

func URL() FieldValidator {
	return func(value interface{}) error {
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string")
		}
		if !ValidateURL(s) {
			return fmt.Errorf("invalid URL")
		}
		return nil
	}
}

func InValues(allowed ...interface{}) FieldValidator {
	return func(value interface{}) error {
		for _, a := range allowed {
			if reflect.DeepEqual(value, a) {
				return nil
			}
		}
		return fmt.Errorf("value must be one of %v", allowed)
	}
}

func (s *Schema) Validate(data map[string]interface{}) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]interface{})
	var errs []string
	for name, field := range s.fields {
		value, ok := data[name]
		if !ok || value == nil {
			for _, alias := range field.Aliases {
				if v, found := data[alias]; found {
					value = v
					ok = true
					break
				}
			}
		}
		if !ok || value == nil {
			if field.Required {
				errs = append(errs, fmt.Sprintf("%s is required", name))
			} else if field.Default != nil {
				result[name] = field.Default
			}
			continue
		}
		converted, err := convertValue(value, field.Type)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		for _, validator := range field.Validators {
			if err := validator(converted); err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", name, err))
			}
		}
		result[name] = converted
	}
	if len(errs) > 0 {
		panic(NewHTTPException(StatusUnprocessableEntity, map[string]interface{}{
			"detail": "Validation failed",
			"errors": errs,
		}))
	}
	for key, value := range data {
		if _, ok := s.fields[key]; !ok {
			result[key] = value
		}
	}
	return result
}

func convertValue(value interface{}, fieldType FieldType) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	switch fieldType {
	case FieldString:
		switch v := value.(type) {
		case string:
			return v, nil
		default:
			return fmt.Sprintf("%v", v), nil
		}
	case FieldInt:
		switch v := value.(type) {
		case int:
			return v, nil
		case float64:
			return int(v), nil
		case string:
			return strconv.Atoi(v)
		case json.Number:
			return strconv.Atoi(string(v))
		default:
			return 0, fmt.Errorf("expected integer")
		}
	case FieldFloat:
		switch v := value.(type) {
		case float64:
			return v, nil
		case int:
			return float64(v), nil
		case string:
			return strconv.ParseFloat(v, 64)
		case json.Number:
			return strconv.ParseFloat(string(v), 64)
		default:
			return 0.0, fmt.Errorf("expected number")
		}
	case FieldBool:
		switch v := value.(type) {
		case bool:
			return v, nil
		case string:
			return strconv.ParseBool(v)
		default:
			return false, fmt.Errorf("expected boolean")
		}
	case FieldTime:
		switch v := value.(type) {
		case time.Time:
			return v, nil
		case string:
			return time.Parse(time.RFC3339, v)
		default:
			return time.Time{}, fmt.Errorf("expected ISO 8601 timestamp")
		}
	case FieldJSON:
		return value, nil
	case FieldSlice:
		val := reflect.ValueOf(value)
		if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
			return value, nil
		}
		return []interface{}{value}, nil
	case FieldMap:
		val := reflect.ValueOf(value)
		if val.Kind() == reflect.Map {
			return value, nil
		}
		return nil, fmt.Errorf("expected object")
	default:
		return value, nil
	}
}

func (s *Schema) JSON(in interface{}, out interface{}) error {
	data, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func (s *Schema) ToJSON(data map[string]interface{}) ([]byte, error) {
	return json.Marshal(data)
}

func (s *Schema) Fields() map[string]*Field {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make(map[string]*Field, len(s.fields))
	for k, v := range s.fields {
		cp[k] = v
	}
	return cp
}

type ObjectSerializer struct {
	schema *Schema
}

func NewObjectSerializer(schema *Schema) *ObjectSerializer {
	return &ObjectSerializer{schema: schema}
}

func (s *ObjectSerializer) Serialize(obj interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *ObjectSerializer) Deserialize(data map[string]interface{}, out interface{}) error {
	validated := s.schema.Validate(data)
	jsonData, err := json.Marshal(validated)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, out)
}

type SerializerFunc func(interface{}) (interface{}, error)

func WrapSerializer(fn SerializerFunc) Serializer {
	return &funcSerializer{fn: fn}
}

type funcSerializer struct {
	fn SerializerFunc
}

func (s *funcSerializer) Serialize(v interface{}) (interface{}, error) {
	return s.fn(v)
}

func (s *funcSerializer) Deserialize(v interface{}, out interface{}) error {
	result, err := s.fn(v)
	if err != nil {
		return err
	}
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

type ListSerializer struct {
	itemSerializer Serializer
}

func NewListSerializer(item Serializer) *ListSerializer {
	return &ListSerializer{itemSerializer: item}
}

func (s *ListSerializer) Serialize(items interface{}) ([]interface{}, error) {
	val := reflect.ValueOf(items)
	if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		return nil, fmt.Errorf("expected slice, got %T", items)
	}
	result := make([]interface{}, val.Len())
	for i := 0; i < val.Len(); i++ {
		serialized, err := s.itemSerializer.Serialize(val.Index(i).Interface())
		if err != nil {
			return nil, fmt.Errorf("item %d: %w", i, err)
		}
		result[i] = serialized
	}
	return result, nil
}

func (s *ListSerializer) Deserialize(data interface{}, out interface{}) error {
	items, ok := data.([]interface{})
	if !ok {
		return fmt.Errorf("expected array")
	}
	sliceType := reflect.TypeOf(out).Elem()
	slice := reflect.MakeSlice(sliceType, len(items), len(items))
	for i, item := range items {
		itemPtr := reflect.New(sliceType.Elem())
		if err := s.itemSerializer.Deserialize(item, itemPtr.Interface()); err != nil {
			return fmt.Errorf("item %d: %w", i, err)
		}
		slice.Index(i).Set(itemPtr.Elem())
	}
	reflect.ValueOf(out).Elem().Set(slice)
	return nil
}

type SerializerRegistry struct {
	mu         sync.RWMutex
	serializers map[string]Serializer
}

func NewSerializerRegistry() *SerializerRegistry {
	return &SerializerRegistry{
		serializers: make(map[string]Serializer),
	}
}

func (r *SerializerRegistry) Register(name string, s Serializer) {
	r.mu.Lock()
	r.serializers[name] = s
	r.mu.Unlock()
}

func (r *SerializerRegistry) Get(name string) (Serializer, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.serializers[name]
	return s, ok
}

func (r *SerializerRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.serializers))
	for n := range r.serializers {
		names = append(names, n)
	}
	return names
}

var defaultRegistry = NewSerializerRegistry()

func RegisterSerializer(name string, s Serializer) {
	defaultRegistry.Register(name, s)
}

func GetSerializer(name string) (Serializer, bool) {
	return defaultRegistry.Get(name)
}

type ModelSerializer struct {
	Model interface{}
	Schema *Schema
}

func NewModelSerializer(model interface{}) *ModelSerializer {
	s := &ModelSerializer{
		Model:  model,
		Schema: NewSchema(),
	}
	s.buildFromModel()
	return s
}

func (s *ModelSerializer) buildFromModel() {
	t := reflect.TypeOf(s.Model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		jsonTag := field.Tag.Get("json")
		name := strings.Split(jsonTag, ",")[0]
		if name == "" {
			name = toSnakeCase(field.Name)
		}
		fieldType := goTypeToFieldType(field.Type)
		s.Schema.Field(name, fieldType)
	}
}

func goTypeToFieldType(t reflect.Type) FieldType {
	switch t.Kind() {
	case reflect.String:
		return FieldString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return FieldInt
	case reflect.Float32, reflect.Float64:
		return FieldFloat
	case reflect.Bool:
		return FieldBool
	case reflect.Slice, reflect.Array:
		return FieldSlice
	case reflect.Map:
		return FieldMap
	case reflect.Struct:
		if t == reflect.TypeOf(time.Time{}) {
			return FieldTime
		}
		return FieldJSON
	default:
		return FieldAny
	}
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(r + 32)
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func regexMatch(pattern, text string) bool {
	return ReMatch(pattern, text)
}
