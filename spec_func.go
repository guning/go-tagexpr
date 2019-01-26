package tagexpr

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"unicode"
)

// --------------------------- Built-in function ---------------------------

type lenFnExprNode struct{ exprBackground }

func (p *Expr) readLenFnExprNode(expr *string) ExprNode {
	if !strings.HasPrefix(*expr, "len(") {
		return nil
	}
	*expr = (*expr)[3:]
	lastStr := *expr
	s := strings.TrimLeftFunc((*expr)[1:], unicode.IsSpace)
	if strings.HasPrefix(s, ")") {
		*expr = "($" + s
	}
	operand, subExprNode := readGroupExprNode(expr)
	if operand == nil {
		return nil
	}
	_, err := p.parseExprNode(subExprNode, operand)
	if err != nil {
		*expr = lastStr
		return nil
	}
	e := &lenFnExprNode{}
	e.SetRightOperand(operand)
	return e
}

func (le *lenFnExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	param := le.rightOperand.Run(currField, tagExpr)
	switch v := param.(type) {
	case string:
		return float64(len(v))
	case float64, bool:
		return nil
	}
	defer func() { recover() }()
	v := reflect.ValueOf(param)
	return float64(v.Len())
}

type regexpFnExprNode struct {
	exprBackground
	re *regexp.Regexp
}

func (p *Expr) readRegexpFnExprNode(expr *string) ExprNode {
	if !strings.HasPrefix(*expr, "regexp(") {
		return nil
	}
	*expr = (*expr)[6:]
	lastStr := *expr
	subExprNode := readPairedSymbol(expr, '(', ')')
	if subExprNode == nil {
		return nil
	}
	s := readPairedSymbol(trimLeftSpace(subExprNode), '\'', '\'')
	if s == nil {
		*expr = lastStr
		return nil
	}
	rege, err := regexp.Compile(*s)
	if err != nil {
		*expr = lastStr
		return nil
	}
	operand := newGroupExprNode()
	trimLeftSpace(subExprNode)
	if strings.HasPrefix(*subExprNode, ",") {
		*subExprNode = (*subExprNode)[1:]
		_, err = p.parseExprNode(trimLeftSpace(subExprNode), operand)
		if err != nil {
			*expr = lastStr
			return nil
		}
	} else {
		var currFieldVal = "$"
		p.parseExprNode(&currFieldVal, operand)
	}
	trimLeftSpace(subExprNode)
	if *subExprNode != "" {
		*expr = lastStr
		return nil
	}
	e := &regexpFnExprNode{
		re: rege,
	}
	e.SetRightOperand(operand)
	return e
}

func (re *regexpFnExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	param := re.rightOperand.Run(currField, tagExpr)
	switch v := param.(type) {
	case string:
		return re.re.MatchString(v)
	case float64, bool:
		return nil
	}
	v := reflect.ValueOf(param)
	if v.Kind() == reflect.String {
		return re.re.MatchString(v.String())
	}
	return nil
}

type sprintfFnExprNode struct {
	exprBackground
	format string
	args   []ExprNode
}

func (p *Expr) readSprintfFnExprNode(expr *string) ExprNode {
	if !strings.HasPrefix(*expr, "sprintf(") {
		return nil
	}
	*expr = (*expr)[7:]
	lastStr := *expr
	subExprNode := readPairedSymbol(expr, '(', ')')
	if subExprNode == nil {
		return nil
	}
	format := readPairedSymbol(trimLeftSpace(subExprNode), '\'', '\'')
	if format == nil {
		*expr = lastStr
		return nil
	}
	e := &sprintfFnExprNode{
		format: *format,
	}
	for {
		trimLeftSpace(subExprNode)
		if len(*subExprNode) == 0 {
			return e
		}
		if strings.HasPrefix(*subExprNode, ",") {
			*subExprNode = (*subExprNode)[1:]
			operand := newGroupExprNode()
			_, err := p.parseExprNode(trimLeftSpace(subExprNode), operand)
			if err != nil {
				*expr = lastStr
				return nil
			}
			sortPriority(operand.RightOperand())
			e.args = append(e.args, operand)
		} else {
			*expr = lastStr
			return nil
		}
	}
}

func (se *sprintfFnExprNode) Run(currField string, tagExpr *TagExpr) interface{} {
	var args []interface{}
	if n := len(se.args); n > 0 {
		args = make([]interface{}, n)
		for i, e := range se.args {
			args[i] = e.Run(currField, tagExpr)
		}
	}
	return fmt.Sprintf(se.format, args...)
}
