package template

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/withgalaxy/galaxy/pkg/executor"
)

type Engine struct {
	ctx       *executor.Context
	parentCtx *executor.Context
}

func NewEngine(ctx *executor.Context) *Engine {
	return &Engine{ctx: ctx}
}

var (
	expressionRegex = regexp.MustCompile(`\{([^}]+)\}`)
	attrRegex       = regexp.MustCompile(`(\w+)=\{([^}]+)\}|(\w+)="([^"]+)"|(\w+)='([^']+)'|(\w+)`)
)

type RenderOptions struct {
	Props     map[string]interface{}
	Slots     map[string]string
	ParentCtx *executor.Context
}

func (e *Engine) Render(template string, opts *RenderOptions) (string, error) {
	if opts != nil {
		for k, v := range opts.Props {
			e.ctx.SetProp(k, v)
			e.ctx.Set(k, v)
		}
		if opts.Slots != nil {
			e.ctx.Slots = opts.Slots
		}
		if opts.ParentCtx != nil {
			e.parentCtx = opts.ParentCtx
		}
	}

	result := template

	result = e.renderDirectives(result)
	result = e.renderSlots(result)

	if e.parentCtx != nil {
		oldCtx := e.ctx
		e.ctx = e.parentCtx
		result = e.renderDirectives(result)
		e.ctx = oldCtx
	} else {
		result = e.renderDirectives(result)
	}

	result = e.renderHtmlExpressions(result)
	result = e.renderExpressions(result)

	return result, nil
}

func (e *Engine) renderHtmlExpressions(template string) string {
	htmlRegex := regexp.MustCompile(`\{@html\s+([^}]+)\}`)
	return htmlRegex.ReplaceAllStringFunc(template, func(match string) string {
		matches := htmlRegex.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}

		varName := strings.TrimSpace(matches[1])

		if val, ok := e.ctx.Get(varName); ok {
			return fmt.Sprintf("%v", val)
		}

		if val, ok := e.ctx.GetProp(varName); ok {
			return fmt.Sprintf("%v", val)
		}

		return match
	})
}

func (e *Engine) renderExpressions(template string) string {
	return expressionRegex.ReplaceAllStringFunc(template, func(match string) string {
		expr := strings.Trim(match, "{}")
		expr = strings.TrimSpace(expr)

		// Skip @html expressions (already processed)
		if strings.HasPrefix(expr, "@html") {
			return match
		}

		if val, ok := e.ctx.Get(expr); ok {
			return fmt.Sprintf("%v", val)
		}

		if val, ok := e.ctx.GetProp(expr); ok {
			return fmt.Sprintf("%v", val)
		}

		if strings.Contains(expr, ".") {
			result, ok := e.evaluateExpression(expr)
			if ok {
				return result
			}
		}

		return match
	})
}

func (e *Engine) renderSlots(template string) string {
	slotRegex := regexp.MustCompile(`<slot(?:\s+name="(\w+)")?\s*/>|<slot(?:\s+name="(\w+)")?>.*?</slot>`)

	return slotRegex.ReplaceAllStringFunc(template, func(match string) string {
		matches := slotRegex.FindStringSubmatch(match)

		var slotName string
		if matches[1] != "" {
			slotName = matches[1]
		} else if matches[2] != "" {
			slotName = matches[2]
		} else {
			slotName = "default"
		}

		if content, ok := e.ctx.Slots[slotName]; ok {
			return content
		}

		innerRegex := regexp.MustCompile(`<slot[^>]*>(.*?)</slot>`)
		if innerMatch := innerRegex.FindStringSubmatch(match); len(innerMatch) > 1 {
			return innerMatch[1]
		}

		return ""
	})
}

func (e *Engine) SetParentContext(parentCtx *executor.Context) {
	e.parentCtx = parentCtx
}

func (e *Engine) GetContextForSlots() *executor.Context {
	if e.parentCtx != nil {
		return e.parentCtx
	}
	return e.ctx
}

func findDirectiveElement(template string, directiveName string) (tag string, attrs string, content string, start int, end int, found bool) {
	directiveAttr := directiveName + "="

	idx := strings.Index(template, directiveAttr)
	if idx == -1 {
		return "", "", "", 0, 0, false
	}

	openStart := strings.LastIndex(template[:idx], "<")
	if openStart == -1 {
		return "", "", "", 0, 0, false
	}

	// Find tag closing > by tracking brace depth
	tagNameEnd := openStart + 1
	for tagNameEnd < len(template) && template[tagNameEnd] != ' ' && template[tagNameEnd] != '>' {
		tagNameEnd++
	}

	pos := tagNameEnd
	braceDepth := 0
	var openEnd int
	foundOpen := false

	for pos < len(template) {
		c := template[pos]

		if c == '{' {
			braceDepth++
		} else if c == '}' {
			braceDepth--
		} else if c == '>' && braceDepth == 0 {
			openEnd = pos
			foundOpen = true
			break
		}

		pos++
	}

	if !foundOpen {
		return "", "", "", 0, 0, false
	}

	tag = template[openStart+1 : tagNameEnd]
	attrs = strings.TrimSpace(template[tagNameEnd:openEnd])

	depth := 1
	searchPos := openEnd + 1
	searchTag := "<" + tag
	closeTag := "</" + tag + ">"

	for searchPos < len(template) && depth > 0 {
		nextOpen := strings.Index(template[searchPos:], searchTag)
		nextClose := strings.Index(template[searchPos:], closeTag)

		if nextClose == -1 {
			return "", "", "", 0, 0, false
		}

		if nextOpen != -1 && nextOpen < nextClose {
			if searchPos+nextOpen+len(searchTag) < len(template) {
				nextChar := template[searchPos+nextOpen+len(searchTag)]
				if nextChar == ' ' || nextChar == '>' {
					depth++
					searchPos += nextOpen + len(searchTag)
					continue
				}
			}
			searchPos += nextOpen + 1
			continue
		}

		depth--
		if depth == 0 {
			content = template[openEnd+1 : searchPos+nextClose]
			end = searchPos + nextClose + len(closeTag)
			return tag, attrs, content, openStart, end, true
		}
		searchPos += nextClose + len(closeTag)
	}

	return "", "", "", 0, 0, false
}

func replaceAllDirectives(template string, directive string, replacer func(tag, attrs, content string) string) string {
	result := template
	offset := 0

	for {
		tag, attrs, content, start, end, found := findDirectiveElement(result[offset:], directive)
		if !found {
			break
		}

		start += offset
		end += offset

		replacement := replacer(tag, attrs, content)
		result = result[:start] + replacement + result[end:]
		offset = start + len(replacement)
	}

	return result
}

func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func findNextSiblingElement(template string, startPos int) (tag string, attrs string, content string, start int, end int, found bool) {
	pos := startPos
	for pos < len(template) && isWhitespace(template[pos]) {
		pos++
	}

	if pos >= len(template) || template[pos] != '<' {
		return "", "", "", 0, 0, false
	}

	if pos+1 < len(template) && (template[pos+1] == '/' || template[pos+1] == '!') {
		return "", "", "", 0, 0, false
	}

	tagStart := pos + 1
	tagEnd := tagStart
	for tagEnd < len(template) && template[tagEnd] != ' ' && template[tagEnd] != '>' {
		tagEnd++
	}
	tag = template[tagStart:tagEnd]

	// Find opening tag end by tracking brace depth
	// This handles comparison operators like >= inside {...}
	attrPos := tagEnd
	braceDepth := 0
	var openEnd int
	foundOpen := false

	for attrPos < len(template) {
		c := template[attrPos]

		if c == '{' {
			braceDepth++
		} else if c == '}' {
			braceDepth--
		} else if c == '>' && braceDepth == 0 {
			openEnd = attrPos
			foundOpen = true
			break
		}

		attrPos++
	}

	if !foundOpen {
		return "", "", "", 0, 0, false
	}

	attrs = strings.TrimSpace(template[tagEnd:openEnd])

	depth := 1
	searchPos := openEnd + 1
	searchTag := "<" + tag
	closeTag := "</" + tag + ">"

	for searchPos < len(template) && depth > 0 {
		nextOpen := strings.Index(template[searchPos:], searchTag)
		nextClose := strings.Index(template[searchPos:], closeTag)

		if nextClose == -1 {
			return "", "", "", 0, 0, false
		}

		if nextOpen != -1 && nextOpen < nextClose {
			if searchPos+nextOpen+len(searchTag) < len(template) {
				nextChar := template[searchPos+nextOpen+len(searchTag)]
				if nextChar == ' ' || nextChar == '>' {
					depth++
					searchPos += nextOpen + len(searchTag)
					continue
				}
			}
			searchPos += nextOpen + 1
			continue
		}

		depth--
		if depth == 0 {
			content = template[openEnd+1 : searchPos+nextClose]
			end = searchPos + nextClose + len(closeTag)
			return tag, attrs, content, pos, end, true
		}
		searchPos += nextClose + len(closeTag)
	}

	return "", "", "", 0, 0, false
}

type ConditionalBranch struct {
	Type      string
	Condition string
	Tag       string
	Attrs     string
	Content   string
	Start     int
	End       int
}

func findConditionalBlock(template string, directive string) (branches []ConditionalBranch, blockStart int, blockEnd int, found bool) {
	ifTag, ifAttrs, ifContent, ifStart, ifEnd, ifFound := findDirectiveElement(template, directive)
	if !ifFound {
		return nil, 0, 0, false
	}

	condStart := strings.Index(ifAttrs, directive+"={")
	if condStart == -1 {
		return nil, 0, 0, false
	}
	condStart += len(directive + "={")
	condEnd := strings.Index(ifAttrs[condStart:], "}")
	if condEnd == -1 {
		return nil, 0, 0, false
	}
	condition := strings.TrimSpace(ifAttrs[condStart : condStart+condEnd])

	otherAttrs := strings.TrimSpace(
		ifAttrs[:condStart-len(directive+"={")] +
			ifAttrs[condStart+condEnd+1:],
	)

	branches = []ConditionalBranch{{
		Type:      "if",
		Condition: condition,
		Tag:       ifTag,
		Attrs:     otherAttrs,
		Content:   ifContent,
		Start:     ifStart,
		End:       ifEnd,
	}}

	blockStart = ifStart
	blockEnd = ifEnd

	currentEnd := ifEnd
	for {
		sibTag, sibAttrs, sibContent, sibStart, sibEnd, sibFound := findNextSiblingElement(template, currentEnd)
		if !sibFound {
			break
		}

		if strings.Contains(sibAttrs, "galaxy:elsif={") {
			elsifCondStart := strings.Index(sibAttrs, "galaxy:elsif={")
			elsifCondStart += len("galaxy:elsif={")
			elsifCondEnd := strings.Index(sibAttrs[elsifCondStart:], "}")
			if elsifCondEnd == -1 {
				break
			}
			elsifCondition := strings.TrimSpace(sibAttrs[elsifCondStart : elsifCondStart+elsifCondEnd])

			elsifOtherAttrs := strings.TrimSpace(
				sibAttrs[:elsifCondStart-len("galaxy:elsif={")] +
					sibAttrs[elsifCondStart+elsifCondEnd+1:],
			)

			branches = append(branches, ConditionalBranch{
				Type:      "elsif",
				Condition: elsifCondition,
				Tag:       sibTag,
				Attrs:     elsifOtherAttrs,
				Content:   sibContent,
				Start:     sibStart,
				End:       sibEnd,
			})

			blockEnd = sibEnd
			currentEnd = sibEnd
			continue
		}

		if strings.Contains(sibAttrs, "galaxy:else") {
			elseOtherAttrs := strings.ReplaceAll(sibAttrs, "galaxy:else", "")
			elseOtherAttrs = strings.TrimSpace(elseOtherAttrs)

			branches = append(branches, ConditionalBranch{
				Type:      "else",
				Condition: "",
				Tag:       sibTag,
				Attrs:     elseOtherAttrs,
				Content:   sibContent,
				Start:     sibStart,
				End:       sibEnd,
			})

			blockEnd = sibEnd
			break
		}

		break
	}

	return branches, blockStart, blockEnd, true
}

func (e *Engine) renderDirectives(template string) string {
	template = e.renderIfDirective(template)
	template = e.renderForDirective(template)
	return template
}

func (e *Engine) renderIfDirective(template string) string {
	result := template
	offset := 0

	for {
		branches, blockStart, blockEnd, found := findConditionalBlock(result[offset:], "galaxy:if")
		if !found {
			break
		}

		blockStart += offset
		blockEnd += offset

		var renderBranch *ConditionalBranch
		for i := range branches {
			branch := &branches[i]

			if branch.Type == "if" || branch.Type == "elsif" {
				if e.evaluateCondition(branch.Condition) {
					renderBranch = branch
					break
				}
			} else if branch.Type == "else" {
				renderBranch = branch
				break
			}
		}

		var replacement string
		if renderBranch != nil {
			processedContent := e.renderDirectives(renderBranch.Content)

			if renderBranch.Attrs != "" {
				replacement = fmt.Sprintf("<%s %s>%s</%s>",
					renderBranch.Tag,
					renderBranch.Attrs,
					processedContent,
					renderBranch.Tag)
			} else {
				replacement = fmt.Sprintf("<%s>%s</%s>",
					renderBranch.Tag,
					processedContent,
					renderBranch.Tag)
			}
		} else {
			replacement = ""
		}

		result = result[:blockStart] + replacement + result[blockEnd:]
		offset = blockStart + len(replacement)
	}

	return result
}

func (e *Engine) renderForDirective(template string) string {
	return replaceAllDirectives(template, "galaxy:for", func(tag, attrs, content string) string {
		forStart := strings.Index(attrs, "galaxy:for={")
		if forStart == -1 {
			return fmt.Sprintf("<%s %s>%s</%s>", tag, attrs, content, tag)
		}

		forStart += len("galaxy:for={")
		forEnd := strings.Index(attrs[forStart:], "}")
		if forEnd == -1 {
			return fmt.Sprintf("<%s %s>%s</%s>", tag, attrs, content, tag)
		}

		loopExpr := strings.TrimSpace(attrs[forStart : forStart+forEnd])

		otherAttrs := strings.TrimSpace(
			attrs[:forStart-len("galaxy:for={")] +
				attrs[forStart+forEnd+1:],
		)

		parts := strings.Fields(loopExpr)
		if len(parts) < 3 || parts[1] != "in" {
			return fmt.Sprintf("<%s %s>%s</%s>", tag, attrs, content, tag)
		}

		itemVar := parts[0]
		arrayName := parts[2]

		if val, ok := e.ctx.Get(arrayName); ok {
			var items []interface{}

			switch v := val.(type) {
			case []interface{}:
				items = v
			case []string:
				items = make([]interface{}, len(v))
				for i, s := range v {
					items[i] = s
				}
			case []int:
				items = make([]interface{}, len(v))
				for i, n := range v {
					items[i] = n
				}
			default:
				// Use reflection to handle any slice type ([]MyStruct, []*MyStruct, etc.)
				rv := reflect.ValueOf(val)
				if rv.Kind() == reflect.Slice {
					items = make([]interface{}, rv.Len())
					for i := 0; i < rv.Len(); i++ {
						items[i] = rv.Index(i).Interface()
					}
				} else {
					return fmt.Sprintf("<%s %s>%s</%s>", tag, attrs, content, tag)
				}
			}

			if len(items) > 0 {
				var result strings.Builder

				for _, item := range items {
					oldVal, hadOld := e.ctx.Get(itemVar)
					e.ctx.Set(itemVar, item)

					rendered := e.renderExpressions(content)

					if otherAttrs != "" {
						result.WriteString(fmt.Sprintf("<%s %s>%s</%s>", tag, otherAttrs, rendered, tag))
					} else {
						result.WriteString(fmt.Sprintf("<%s>%s</%s>", tag, rendered, tag))
					}

					if hadOld {
						e.ctx.Set(itemVar, oldVal)
					}
				}

				return result.String()
			}
		}

		return fmt.Sprintf("<%s %s>%s</%s>", tag, attrs, content, tag)
	})
}

func (e *Engine) evaluateCondition(condition string) bool {
	condition = strings.TrimSpace(condition)

	// Check two-char operators first (order matters!)
	if strings.Contains(condition, "==") {
		parts := strings.SplitN(condition, "==", 2)
		left := e.evaluateValue(strings.TrimSpace(parts[0]))
		right := e.evaluateValue(strings.TrimSpace(parts[1]))
		return e.compareEqual(left, right)
	}

	if strings.Contains(condition, "!=") {
		parts := strings.SplitN(condition, "!=", 2)
		left := e.evaluateValue(strings.TrimSpace(parts[0]))
		right := e.evaluateValue(strings.TrimSpace(parts[1]))
		return !e.compareEqual(left, right)
	}

	if strings.Contains(condition, ">=") {
		parts := strings.SplitN(condition, ">=", 2)
		left := e.evaluateValue(strings.TrimSpace(parts[0]))
		right := e.evaluateValue(strings.TrimSpace(parts[1]))
		cmp := e.compareValues(left, right)
		return cmp >= 0
	}

	if strings.Contains(condition, "<=") {
		parts := strings.SplitN(condition, "<=", 2)
		left := e.evaluateValue(strings.TrimSpace(parts[0]))
		right := e.evaluateValue(strings.TrimSpace(parts[1]))
		cmp := e.compareValues(left, right)
		return cmp <= 0
	}

	// Check single-char operators after two-char
	if strings.Contains(condition, ">") {
		parts := strings.SplitN(condition, ">", 2)
		left := e.evaluateValue(strings.TrimSpace(parts[0]))
		right := e.evaluateValue(strings.TrimSpace(parts[1]))
		cmp := e.compareValues(left, right)
		return cmp > 0
	}

	if strings.Contains(condition, "<") {
		parts := strings.SplitN(condition, "<", 2)
		left := e.evaluateValue(strings.TrimSpace(parts[0]))
		right := e.evaluateValue(strings.TrimSpace(parts[1]))
		cmp := e.compareValues(left, right)
		return cmp < 0
	}

	// Simple variable lookup
	val := e.evaluateValue(condition)
	return e.isTruthy(val)
}

func (e *Engine) evaluateValue(expr string) interface{} {
	expr = strings.TrimSpace(expr)

	// String literal
	if (strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"")) ||
		(strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'")) {
		return expr[1 : len(expr)-1]
	}

	// Number literal
	if len(expr) > 0 && (expr[0] >= '0' && expr[0] <= '9') || expr[0] == '-' {
		if val, err := fmt.Sscanf(expr, "%d", new(int64)); err == nil && val == 1 {
			var num int64
			fmt.Sscanf(expr, "%d", &num)
			return num
		}
		if val, err := fmt.Sscanf(expr, "%f", new(float64)); err == nil && val == 1 {
			var num float64
			fmt.Sscanf(expr, "%f", &num)
			return num
		}
	}

	// Variable lookup
	if val, ok := e.ctx.Get(expr); ok {
		return val
	}

	return nil
}

func (e *Engine) isTruthy(val interface{}) bool {
	if val == nil {
		return false
	}

	switch v := val.(type) {
	case bool:
		return v
	case int64:
		return v != 0
	case float64:
		return v != 0
	case string:
		return v != ""
	case []interface{}:
		return len(v) > 0
	default:
		return true
	}
}

func (e *Engine) compareEqual(left, right interface{}) bool {
	if left == nil || right == nil {
		return left == right
	}

	// Same type comparison
	switch l := left.(type) {
	case string:
		if r, ok := right.(string); ok {
			return l == r
		}
	case int64:
		if r, ok := right.(int64); ok {
			return l == r
		}
		if r, ok := right.(float64); ok {
			return float64(l) == r
		}
	case float64:
		if r, ok := right.(float64); ok {
			return l == r
		}
		if r, ok := right.(int64); ok {
			return l == float64(r)
		}
	case bool:
		if r, ok := right.(bool); ok {
			return l == r
		}
	}

	return false
}

func (e *Engine) compareValues(left, right interface{}) int {
	// Compare numerics
	leftNum := e.toNumber(left)
	rightNum := e.toNumber(right)

	if leftNum < rightNum {
		return -1
	}
	if leftNum > rightNum {
		return 1
	}
	return 0
}

func (e *Engine) toNumber(val interface{}) float64 {
	switch v := val.(type) {
	case int64:
		return float64(v)
	case float64:
		return v
	case string:
		var num float64
		fmt.Sscanf(v, "%f", &num)
		return num
	default:
		return 0
	}
}

func ParseAttributes(attrString string) map[string]interface{} {
	attrs := make(map[string]interface{})

	matches := attrRegex.FindAllStringSubmatch(attrString, -1)
	for _, match := range matches {
		if match[1] != "" {
			attrs[match[1]] = match[2]
		} else if match[3] != "" {
			attrs[match[3]] = match[4]
		} else if match[5] != "" {
			attrs[match[5]] = match[6]
		} else if match[7] != "" {
			attrs[match[7]] = true
		}
	}

	return attrs
}

func (e *Engine) evaluateExpression(expr string) (string, bool) {
	parts := strings.Split(expr, ".")
	if len(parts) < 2 {
		return "", false
	}

	varName := parts[0]
	val, ok := e.ctx.Get(varName)
	if !ok {
		return "", false
	}

	methodCall := parts[1]

	// Check if it's a method call (has parentheses)
	if strings.Contains(methodCall, "(") && strings.HasSuffix(methodCall, ")") {
		// Extract method name and arguments
		openParen := strings.Index(methodCall, "(")
		methodName := methodCall[:openParen]
		argsStr := methodCall[openParen+1 : len(methodCall)-1]

		// Try reflection-based method invocation
		v := reflect.ValueOf(val)
		method := v.MethodByName(methodName)
		if method.IsValid() {
			// Parse arguments
			var args []reflect.Value
			if argsStr != "" {
				// Simple string argument parsing (quoted strings)
				argParts := strings.Split(argsStr, ",")
				for _, arg := range argParts {
					arg = strings.TrimSpace(arg)
					// Remove quotes
					arg = strings.Trim(arg, "\"'")
					args = append(args, reflect.ValueOf(arg))
				}
			}

			// Call method
			results := method.Call(args)
			if len(results) > 0 {
				return fmt.Sprintf("%v", results[0].Interface()), true
			}
		}
	} else if strings.HasSuffix(methodCall, "()") {
		methodName := strings.TrimSuffix(methodCall, "()")

		if reqCtx, ok := val.(interface {
			Path() string
			Method() string
			URL() string
		}); ok {
			switch methodName {
			case "Path":
				return reqCtx.Path(), true
			case "Method":
				return reqCtx.Method(), true
			case "URL":
				return reqCtx.URL(), true
			}
		}
	} else {
		if m, ok := val.(map[string]interface{}); ok {
			property := parts[1]
			if propVal, ok := m[property]; ok {
				return fmt.Sprintf("%v", propVal), true
			}
		}
		if m, ok := val.(map[string]any); ok {
			property := parts[1]
			if propVal, ok := m[property]; ok {
				return fmt.Sprintf("%v", propVal), true
			}
		}

		// Try reflection for struct fields (handle multi-level: project.Owner.Name)
		v := reflect.ValueOf(val)

		// Navigate through all field parts (parts[1], parts[2], etc.)
		for i := 1; i < len(parts); i++ {
			// Dereference pointers
			for v.Kind() == reflect.Ptr {
				if v.IsNil() {
					return "", false
				}
				v = v.Elem()
			}

			if v.Kind() == reflect.Struct {
				field := v.FieldByName(parts[i])
				if !field.IsValid() {
					return "", false
				}
				v = field
			} else {
				return "", false
			}
		}

		return fmt.Sprintf("%v", v.Interface()), true
	}

	return "", false
}
