package content

import "time"

type CollectionType string

const (
	CollectionTypeContent CollectionType = "content"
	CollectionTypeData    CollectionType = "data"
)

type Entry struct {
	ID         string
	Slug       string
	Collection string
	Data       map[string]interface{}
	Body       string
	FilePath   string
	RawContent string
}

type CollectionConfig struct {
	Type   CollectionType
	Schema Schema
}

type Schema struct {
	Fields map[string]FieldType
}

type FieldType struct {
	Type     string
	Required bool
}

type BlogPost struct {
	Title       string
	Description string
	PubDate     time.Time
	Author      string
	Tags        []string
	Draft       bool
}

func (e *Entry) GetString(key string) string {
	if val, ok := e.Data[key].(string); ok {
		return val
	}
	return ""
}

func (e *Entry) GetInt(key string) int {
	if val, ok := e.Data[key].(int); ok {
		return val
	}
	return 0
}

func (e *Entry) GetBool(key string) bool {
	if val, ok := e.Data[key].(bool); ok {
		return val
	}
	return false
}

func (e *Entry) GetStringSlice(key string) []string {
	if val, ok := e.Data[key].([]interface{}); ok {
		result := make([]string, 0, len(val))
		for _, v := range val {
			if str, ok := v.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	return nil
}
