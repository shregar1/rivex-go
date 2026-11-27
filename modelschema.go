package rivex

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
)

type ExtraFieldPolicy int

const (
	ExtraIgnore ExtraFieldPolicy = iota
	ExtraAllow
	ExtraForbid
)

type ModelConfig struct {
	Frozen             bool
	Extra              ExtraFieldPolicy
	PopulateByName     bool
	ValidateAssignment bool
	StrStripWhitespace bool
}

type FieldValidatorFunc func(value interface{}, ctx *ValidationContext) error
type ModelValidatorFunc func(values map[string]interface{}, ctx *ValidationContext) error
type FieldSerializerFunc func(value interface{}) interface{}
type ComputedFieldFunc func(values map[string]interface{}) interface{}

type fieldDefinition struct {
	Name         string
	Alias        string
	Aliases      []string
	Type         reflect.Type
	Required     bool
	Default      interface{}
	Validators   []FieldValidatorFunc
	Serializer   FieldSerializerFunc
	IsComputed   bool
	ComputedFunc ComputedFieldFunc
	IsExcluded   bool
	Tag          string
}

type ValidationContext struct {
	Schema interface{}
	Values map[string]interface{}
	Errors []ValidationError
	mu     sync.Mutex
}

func (c *ValidationContext) AddError(field string, err error) {
	c.mu.Lock()
	c.Errors = append(c.Errors, ValidationError{Field: field, Error: err.Error()})
	c.mu.Unlock()
}

func (c *ValidationContext) Error() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	msgs := make([]string, len(c.Errors))
	for i, e := range c.Errors {
		msgs[i] = fmt.Sprintf("%s: %s", e.Field, e.Error)
	}
	return strings.Join(msgs, "; ")
}

func (c *ValidationContext) ErrorMap() map[string][]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make(map[string][]string)
	for _, e := range c.Errors {
		result[e.Field] = append(result[e.Field], e.Error)
	}
	return result
}

type ValidationError struct {
	Field string
	Error string
}

func (e ValidationError) String() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Error)
}

type ValidationErrors struct {
	Context *ValidationContext
}

func (e *ValidationErrors) Error() string {
	return "modelschema: " + e.Context.Error()
}

func (e *ValidationErrors) Errors() []ValidationError {
	return e.Context.Errors
}

func (e *ValidationErrors) As(target interface{}) bool {
	if t, ok := target.(**ValidationErrors); ok {
		*t = e
		return true
	}
	return false
}

type ModelSchema[T any] struct {
	modelType       reflect.Type
	config          ModelConfig
	fields          map[string]*fieldDefinition
	orderedFields   []string
	modelValidators []ModelValidatorFunc
	computedFields  map[string]ComputedFieldFunc
	mu              sync.RWMutex
	validator       *validator.Validate
}

func NewModelSchema[T any]() *ModelSchema[T] {
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil {
		t = reflect.TypeOf((*T)(nil)).Elem()
	}
	s := &ModelSchema[T]{
		modelType:      t,
		config:         ModelConfig{Extra: ExtraIgnore},
		fields:         make(map[string]*fieldDefinition),
		computedFields: make(map[string]ComputedFieldFunc),
		validator:      newValidator(),
	}
	s.parseStruct(t)
	return s
}

func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	_ = v.RegisterValidation("email", func(fl validator.FieldLevel) bool {
		s, ok := fl.Field().Interface().(string)
		if !ok {
			return false
		}
		return ValidateEmail(s)
	})
	_ = v.RegisterValidation("url", func(fl validator.FieldLevel) bool {
		s, ok := fl.Field().Interface().(string)
		if !ok {
			return false
		}
		return ValidateURL(s)
	})
	_ = v.RegisterValidation("min_length", func(fl validator.FieldLevel) bool {
		n, _ := strconv.Atoi(fl.Param())
		s, ok := fl.Field().Interface().(string)
		if !ok {
			return false
		}
		return len(s) >= n
	})
	_ = v.RegisterValidation("max_length", func(fl validator.FieldLevel) bool {
		n, _ := strconv.Atoi(fl.Param())
		s, ok := fl.Field().Interface().(string)
		if !ok {
			return false
		}
		return len(s) <= n
	})
	_ = v.RegisterValidation("multiple_of", func(fl validator.FieldLevel) bool {
		target, _ := strconv.ParseFloat(fl.Param(), 64)
		v := toFloatValue(fl.Field().Interface())
		if v == nil || target == 0 {
			return false
		}
		quotient := *v / target
		return quotient == float64(int64(quotient))
	})
	return v
}

func toFloatValue(v interface{}) *float64 {
	f, ok := toFloat(v)
	if !ok {
		return nil
	}
	return &f
}

func (s *ModelSchema[T]) parseStruct(t reflect.Type) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if f.Anonymous {
			continue
		}
		fd := &fieldDefinition{
			Name:  f.Name,
			Alias: f.Name,
			Type:  f.Type,
			Tag:   f.Tag.Get("validate"),
		}
		jsonTag := f.Tag.Get("json")
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] == "-" {
				fd.IsExcluded = true
			} else if parts[0] != "" {
				fd.Alias = parts[0]
			}
		}
		s.fields[fd.Name] = fd
		s.orderedFields = append(s.orderedFields, fd.Name)
	}
}

func (s *ModelSchema[T]) Config() ModelConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func (s *ModelSchema[T]) SetConfig(cfg ModelConfig) *ModelSchema[T] {
	s.mu.Lock()
	s.config = cfg
	s.mu.Unlock()
	return s
}

func (s *ModelSchema[T]) Frozen(v bool) *ModelSchema[T] {
	s.mu.Lock()
	s.config.Frozen = v
	s.mu.Unlock()
	return s
}

func (s *ModelSchema[T]) Extra(policy ExtraFieldPolicy) *ModelSchema[T] {
	s.mu.Lock()
	s.config.Extra = policy
	s.mu.Unlock()
	return s
}

func (s *ModelSchema[T]) Engine() *validator.Validate {
	return s.validator
}

func (s *ModelSchema[T]) RegisterValidation(name string, fn validator.Func, callValidationEvenIfNull ...bool) error {
	return s.validator.RegisterValidation(name, fn, callValidationEvenIfNull...)
}

func (s *ModelSchema[T]) FieldValidator(name string, fn FieldValidatorFunc) *ModelSchema[T] {
	s.mu.Lock()
	defer s.mu.Unlock()
	if fd, ok := s.findFieldLocked(name); ok {
		fd.Validators = append(fd.Validators, fn)
	}
	return s
}

func (s *ModelSchema[T]) ModelValidator(fn ModelValidatorFunc) *ModelSchema[T] {
	s.mu.Lock()
	s.modelValidators = append(s.modelValidators, fn)
	s.mu.Unlock()
	return s
}

func (s *ModelSchema[T]) FieldSerializer(name string, fn FieldSerializerFunc) *ModelSchema[T] {
	s.mu.Lock()
	defer s.mu.Unlock()
	if fd, ok := s.findFieldLocked(name); ok {
		fd.Serializer = fn
	}
	return s
}

func (s *ModelSchema[T]) ComputedField(name string, fn ComputedFieldFunc) *ModelSchema[T] {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.computedFields[name] = fn
	return s
}

func (s *ModelSchema[T]) Validate(data map[string]interface{}) (T, error) {
	var result T
	val := reflect.ValueOf(&result)
	if val.Kind() != reflect.Ptr {
		return result, fmt.Errorf("modelschema: must pass pointer")
	}
	val = val.Elem()
	if !val.CanSet() {
		return result, fmt.Errorf("modelschema: cannot set value")
	}
	vctx := &ValidationContext{
		Schema: s,
		Values: make(map[string]interface{}),
	}
	for key, value := range data {
		if key == "_discriminator" {
			continue
		}
		s.applyField(val, vctx, key, value)
	}
	for name, fn := range s.computedFields {
		field := val.FieldByName(name)
		if !field.IsValid() || !field.CanSet() {
			continue
		}
		computed := fn(vctx.Values)
		if err := setFieldValue(field, computed); err == nil {
			vctx.Values[name] = computed
		}
	}
	s.runModelValidators(vctx)
	if len(vctx.Errors) > 0 {
		return result, &ValidationErrors{Context: vctx}
	}
	if err := s.validator.Struct(result); err != nil {
		for _, fieldErr := range err.(validator.ValidationErrors) {
			vctx.AddError(strings.ToLower(fieldErr.Field()), fieldErr)
		}
		return result, &ValidationErrors{Context: vctx}
	}
	return result, nil
}

func (s *ModelSchema[T]) ModelValidate(data []byte) (T, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return *new(T), err
	}
	return s.Validate(raw)
}

func (s *ModelSchema[T]) applyField(val reflect.Value, vctx *ValidationContext, key string, rawValue interface{}) {
	fd, ok := s.findFieldByKey(key)
	if !ok {
		switch s.config.Extra {
		case ExtraForbid:
			vctx.AddError(key, fmt.Errorf("extra field not allowed"))
		case ExtraIgnore:
		case ExtraAllow:
			vctx.Values[key] = rawValue
		}
		return
	}
	if fd.IsExcluded {
		return
	}
	coerced, err := s.coerce(fd, rawValue)
	if err != nil {
		vctx.AddError(fd.Alias, err)
		return
	}
	if fd.Required && isZeroValue(coerced) {
		vctx.AddError(fd.Alias, fmt.Errorf("field is required"))
		return
	}
	for _, v := range fd.Validators {
		if err := v(coerced, vctx); err != nil {
			vctx.AddError(fd.Alias, err)
		}
	}
	if hasErrorsFor(vctx, fd.Alias) {
		return
	}
	field := val.FieldByName(fd.Name)
	if !field.IsValid() || !field.CanSet() {
		vctx.Values[fd.Alias] = coerced
		return
	}
	if err := setFieldValue(field, coerced); err != nil {
		vctx.AddError(fd.Alias, err)
		return
	}
	vctx.Values[fd.Alias] = coerced
}

func hasErrorsFor(ctx *ValidationContext, field string) bool {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	for _, e := range ctx.Errors {
		if e.Field == field {
			return true
		}
	}
	return false
}

func (s *ModelSchema[T]) findFieldByKey(key string) (*fieldDefinition, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.findFieldLocked(key)
}

func (s *ModelSchema[T]) findFieldLocked(name string) (*fieldDefinition, bool) {
	if fd, ok := s.fields[name]; ok {
		return fd, true
	}
	for _, fd := range s.fields {
		if strings.EqualFold(fd.Name, name) || fd.Alias == name {
			return fd, true
		}
		for _, alias := range fd.Aliases {
			if alias == name {
				return fd, true
			}
		}
	}
	return nil, false
}

func (s *ModelSchema[T]) coerce(fd *fieldDefinition, raw interface{}) (interface{}, error) {
	if raw == nil {
		return nil, nil
	}
	if fd.Type.Kind() == reflect.String {
		if str, ok := raw.(string); ok {
			return str, nil
		}
		return fmt.Sprintf("%v", raw), nil
	}
	if fd.Type == reflect.TypeOf(time.Time{}) {
		switch v := raw.(type) {
		case time.Time:
			return v, nil
		case string:
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				return nil, fmt.Errorf("invalid time: %w", err)
			}
			return t, nil
		default:
			return nil, fmt.Errorf("cannot convert %T to time.Time", raw)
		}
	}
	return coerceValue(raw, fd.Type)
}

func (s *ModelSchema[T]) runModelValidators(vctx *ValidationContext) {
	s.mu.RLock()
	validators := make([]ModelValidatorFunc, len(s.modelValidators))
	copy(validators, s.modelValidators)
	s.mu.RUnlock()
	for _, fn := range validators {
		if err := fn(vctx.Values, vctx); err != nil {
			vctx.AddError("_model", err)
		}
	}
}

func (s *ModelSchema[T]) ModelDump(instance T) (map[string]interface{}, error) {
	val := reflect.ValueOf(instance)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if !val.IsValid() || val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("modelschema: instance is not a struct")
	}
	s.mu.RLock()
	fields := make([]*fieldDefinition, 0, len(s.orderedFields))
	for _, name := range s.orderedFields {
		if fd, ok := s.fields[name]; ok {
			fields = append(fields, fd)
		}
	}
	s.mu.RUnlock()
	result := make(map[string]interface{})
	for _, fd := range fields {
		field := val.FieldByName(fd.Name)
		if !field.IsValid() {
			continue
		}
		var value interface{}
		if fd.Serializer != nil {
			value = fd.Serializer(field.Interface())
		} else {
			value = field.Interface()
		}
		result[fd.Alias] = value
	}
	for name, fn := range s.computedFields {
		values := collectValues(val)
		result[name] = fn(values)
	}
	return result, nil
}

func collectValues(val reflect.Value) map[string]interface{} {
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	result := make(map[string]interface{})
	t := val.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		jsonTag := f.Tag.Get("json")
		name := f.Name
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" && parts[0] != "-" {
				name = parts[0]
			}
		}
		result[name] = val.Field(i).Interface()
	}
	return result
}

func (s *ModelSchema[T]) ModelJSON(instance T) ([]byte, error) {
	dump, err := s.ModelDump(instance)
	if err != nil {
		return nil, err
	}
	return json.Marshal(dump)
}

func (s *ModelSchema[T]) ModelCopy(instance T) (T, error) {
	data, err := s.ModelDump(instance)
	if err != nil {
		return *new(T), err
	}
	return s.Validate(data)
}

func (s *ModelSchema[T]) Fields() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]string, 0, len(s.orderedFields))
	for _, name := range s.orderedFields {
		if fd, ok := s.fields[name]; ok {
			if !fd.IsExcluded {
				result = append(result, fd.Alias)
			}
		}
	}
	return result
}

func (s *ModelSchema[T]) ModelField(name string) *FieldMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fd, ok := s.findFieldLocked(name)
	if !ok {
		return nil
	}
	return &FieldMeta{
		Name:        fd.Alias,
		Required:    strings.Contains(fd.Tag, "required"),
		HasDefault:  fd.Default != nil,
		IsComputed:  fd.IsComputed,
		Constraints: len(fd.Validators),
	}
}

type FieldMeta struct {
	Name        string
	Required    bool
	HasDefault  bool
	Constraints int
	IsComputed  bool
}

func setFieldValue(field reflect.Value, value interface{}) error {
	if !field.CanSet() {
		return fmt.Errorf("field is not settable")
	}
	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		field.Set(reflect.Zero(field.Type()))
		return nil
	}
	if rv.Type().AssignableTo(field.Type()) {
		field.Set(rv)
		return nil
	}
	if rv.Type().ConvertibleTo(field.Type()) {
		field.Set(rv.Convert(field.Type()))
		return nil
	}
	return fmt.Errorf("cannot assign %T to %s", value, field.Type())
}

func coerceValue(value interface{}, target reflect.Type) (interface{}, error) {
	if value == nil {
		return reflect.Zero(target).Interface(), nil
	}
	rv := reflect.ValueOf(value)
	if rv.Type().AssignableTo(target) {
		return value, nil
	}
	switch target.Kind() {
	case reflect.String:
		return fmt.Sprintf("%v", value), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v := value.(type) {
		case float64:
			return int64(v), nil
		case float32:
			return int64(v), nil
		case string:
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("cannot parse %q as int", v)
			}
			return n, nil
		case json.Number:
			n, err := v.Int64()
			if err != nil {
				return nil, err
			}
			return n, nil
		case bool:
			if v {
				return int64(1), nil
			}
			return int64(0), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := toInt64(value)
		if err != nil {
			return nil, err
		}
		return uint64(n), nil
	case reflect.Float32, reflect.Float64:
		switch v := value.(type) {
		case float64:
			return v, nil
		case string:
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, err
			}
			return f, nil
		}
	case reflect.Bool:
		switch v := value.(type) {
		case bool:
			return v, nil
		case string:
			b, err := strconv.ParseBool(v)
			if err != nil {
				return nil, err
			}
			return b, nil
		}
	}
	if rv.Type().ConvertibleTo(target) {
		return rv.Convert(target).Interface(), nil
	}
	return nil, fmt.Errorf("cannot coerce %T to %s", value, target)
}

func toFloat(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int8:
		return float64(x), true
	case int16:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint8:
		return float64(x), true
	case uint16:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	case string:
		f, err := strconv.ParseFloat(x, 64)
		return f, err == nil
	}
	return 0, false
}

func toInt64(v interface{}) (int64, error) {
	switch x := v.(type) {
	case int:
		return int64(x), nil
	case int8:
		return int64(x), nil
	case int16:
		return int64(x), nil
	case int32:
		return int64(x), nil
	case int64:
		return x, nil
	case uint:
		return int64(x), nil
	case uint8:
		return int64(x), nil
	case uint16:
		return int64(x), nil
	case uint32:
		return int64(x), nil
	case uint64:
		return int64(x), nil
	case float64:
		return int64(x), nil
	case float32:
		return int64(x), nil
	case string:
		return strconv.ParseInt(x, 10, 64)
	}
	return 0, fmt.Errorf("cannot convert %T to int64", v)
}

func isZeroValue(v interface{}) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String:
		return rv.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() == 0
	case reflect.Bool:
		return !rv.Bool()
	case reflect.Slice, reflect.Map, reflect.Array:
		return rv.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return rv.IsNil()
	}
	return false
}

type FieldOptions struct {
	Aliases     []string
	Description string
	Example     interface{}
}

type FieldOpt func(*FieldOptions)

func WithAliases(aliases ...string) FieldOpt {
	return func(o *FieldOptions) { o.Aliases = aliases }
}

func WithModelFieldDescription(desc string) FieldOpt {
	return func(o *FieldOptions) { o.Description = desc }
}

func WithExample(ex interface{}) FieldOpt {
	return func(o *FieldOptions) { o.Example = ex }
}

func (s *ModelSchema[T]) AddField(name string, typeHint interface{}, opts ...FieldOpt) *ModelSchema[T] {
	options := FieldOptions{}
	for _, o := range opts {
		o(&options)
	}
	t := reflect.TypeOf(typeHint)
	fd := &fieldDefinition{
		Name:    name,
		Alias:   name,
		Type:    t,
		Aliases: options.Aliases,
	}
	s.mu.Lock()
	s.fields[name] = fd
	s.orderedFields = append(s.orderedFields, name)
	s.mu.Unlock()
	return s
}

type ModelSchemaBuilder[T any] struct {
	schema *ModelSchema[T]
}

func NewModelSchemaBuilder[T any]() *ModelSchemaBuilder[T] {
	return &ModelSchemaBuilder[T]{schema: NewModelSchema[T]()}
}

func (b *ModelSchemaBuilder[T]) Frozen(v bool) *ModelSchemaBuilder[T] {
	b.schema.Frozen(v)
	return b
}

func (b *ModelSchemaBuilder[T]) Extra(policy ExtraFieldPolicy) *ModelSchemaBuilder[T] {
	b.schema.Extra(policy)
	return b
}

func (b *ModelSchemaBuilder[T]) Validator(name string, fn FieldValidatorFunc) *ModelSchemaBuilder[T] {
	b.schema.FieldValidator(name, fn)
	return b
}

func (b *ModelSchemaBuilder[T]) ModelValidator(fn ModelValidatorFunc) *ModelSchemaBuilder[T] {
	b.schema.ModelValidator(fn)
	return b
}

func (b *ModelSchemaBuilder[T]) Serializer(name string, fn FieldSerializerFunc) *ModelSchemaBuilder[T] {
	b.schema.FieldSerializer(name, fn)
	return b
}

func (b *ModelSchemaBuilder[T]) Computed(name string, fn ComputedFieldFunc) *ModelSchemaBuilder[T] {
	b.schema.ComputedField(name, fn)
	return b
}

func (b *ModelSchemaBuilder[T]) Engine() *validator.Validate {
	return b.schema.Engine()
}

func (b *ModelSchemaBuilder[T]) Build() *ModelSchema[T] {
	return b.schema
}

type DiscriminatedUnion struct {
	mu       sync.RWMutex
	tagName  string
	branches map[string]func(data map[string]interface{}) (interface{}, error)
}

func NewDiscriminatedUnion(tagName string) *DiscriminatedUnion {
	return &DiscriminatedUnion{
		tagName:  tagName,
		branches: make(map[string]func(data map[string]interface{}) (interface{}, error)),
	}
}

func (d *DiscriminatedUnion) Register(tagValue string, fn func(data map[string]interface{}) (interface{}, error)) {
	d.mu.Lock()
	d.branches[tagValue] = fn
	d.mu.Unlock()
}

func (d *DiscriminatedUnion) Resolve(data map[string]interface{}) (interface{}, error) {
	tag, ok := data[d.tagName].(string)
	if !ok {
		return nil, fmt.Errorf("discriminated union: missing tag %s", d.tagName)
	}
	d.mu.RLock()
	fn, ok := d.branches[tag]
	d.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("discriminated union: no branch for tag %s", tag)
	}
	return fn(data)
}

func (d *DiscriminatedUnion) TagName() string { return d.tagName }

type NestedSchemaValidator[T any] struct {
	schema *ModelSchema[T]
}

func NewNestedSchemaValidator[T any]() *NestedSchemaValidator[T] {
	return &NestedSchemaValidator[T]{schema: NewModelSchema[T]()}
}

func (n *NestedSchemaValidator[T]) Schema() *ModelSchema[T] {
	return n.schema
}

func (n *NestedSchemaValidator[T]) Validate(data map[string]interface{}) (T, error) {
	return n.schema.Validate(data)
}

type ValidationReport struct {
	TotalErrors int
	ByField     map[string][]string
	Errors      []ValidationError
}

func (s *ModelSchema[T]) ValidateWithReport(data map[string]interface{}) (T, *ValidationReport) {
	result, err := s.Validate(data)
	report := &ValidationReport{
		ByField: make(map[string][]string),
	}
	if err == nil {
		return result, report
	}
	var verrs *ValidationErrors
	if errors.As(err, &verrs) {
		report.TotalErrors = len(verrs.Errors())
		report.Errors = verrs.Errors()
		for _, e := range verrs.Errors() {
			report.ByField[e.Field] = append(report.ByField[e.Field], e.Error)
		}
	} else {
		report.TotalErrors = 1
		report.Errors = []ValidationError{{Field: "_root", Error: err.Error()}}
	}
	return result, report
}

func ToJSONable(v interface{}) (interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}
