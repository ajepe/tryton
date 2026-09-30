package xml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

type Node struct {
	Name     string
	Attrs    map[string]string
	Children []*Node
	Text     string
}

func NewNode(name string) *Node {
	return &Node{Name: name, Attrs: map[string]string{}}
}

func (n *Node) Attr(name string) string {
	return n.Attrs[name]
}

func (n *Node) SetAttr(name, value string) {
	if n.Attrs == nil {
		n.Attrs = map[string]string{}
	}
	n.Attrs[name] = value
}

func (n *Node) Clone() *Node {
	clone := &Node{Name: n.Name, Attrs: map[string]string{}, Text: n.Text}
	for key, value := range n.Attrs {
		clone.Attrs[key] = value
	}
	for _, child := range n.Children {
		clone.Children = append(clone.Children, child.Clone())
	}
	return clone
}

func ParseArch(source string) (*Node, error) {
	decoder := xml.NewDecoder(strings.NewReader(source))
	var root *Node
	var stack []*Node
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch element := token.(type) {
		case xml.StartElement:
			node := &Node{Name: element.Name.Local, Attrs: map[string]string{}}
			for _, attr := range element.Attr {
				node.Attrs[attr.Name.Local] = attr.Value
			}
			if len(stack) == 0 {
				if root == nil {
					root = node
				} else {
					root.Children = append(root.Children, node)
				}
			} else {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, node)
			}
			stack = append(stack, node)
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].Text += string(element)
			}
		}
	}
	if root == nil {
		return nil, fmt.Errorf("empty view architecture")
	}
	return root, nil
}

func MarshalArch(node *Node) (string, error) {
	var buf bytes.Buffer
	encoder := xml.NewEncoder(&buf)
	if err := encodeNode(encoder, node); err != nil {
		return "", err
	}
	if err := encoder.Flush(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func encodeNode(encoder *xml.Encoder, node *Node) error {
	start := xml.StartElement{Name: xml.Name{Local: node.Name}}
	for key, value := range node.Attrs {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: key}, Value: value})
	}
	if err := encoder.EncodeToken(start); err != nil {
		return err
	}
	if len(node.Children) == 0 && node.Text != "" {
		if err := encoder.EncodeToken(xml.CharData(node.Text)); err != nil {
			return err
		}
	}
	for _, child := range node.Children {
		if err := encodeNode(encoder, child); err != nil {
			return err
		}
	}
	return encoder.EncodeToken(start.End())
}

type XPathExpr struct {
	Tag      string
	Attrs    map[string]string
	Ancestor []XPathExpr
}

func ParseXPath(expr string) (*XPathExpr, error) {
	expr = strings.TrimSpace(expr)
	expr = strings.TrimPrefix(expr, "/")
	expr = strings.TrimPrefix(expr, "/")
	segments := strings.Split(expr, "/")
	if len(segments) == 0 || segments[0] == "" {
		return nil, fmt.Errorf("invalid xpath %q", expr)
	}
	parsed := make([]XPathExpr, 0, len(segments))
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" || segment == "." {
			continue
		}
		part := XPathExpr{Attrs: map[string]string{}}
		if idx := strings.Index(segment, "["); idx >= 0 {
			part.Tag = segment[:idx]
			conditions := segment[idx:]
			conditions = strings.Trim(conditions, "[]")
			for _, condition := range splitAttrs(conditions) {
				if strings.HasPrefix(strings.TrimSpace(condition), "@") {
					condition = strings.TrimSpace(condition[1:])
					pair := strings.SplitN(condition, "=", 2)
					if len(pair) == 2 {
						part.Attrs[strings.TrimSpace(pair[0])] = trimQuotes(strings.TrimSpace(pair[1]))
					}
				}
			}
		} else {
			part.Tag = segment
		}
		parsed = append(parsed, part)
	}
	if len(parsed) == 0 {
		return nil, fmt.Errorf("invalid xpath %q", expr)
	}
	exprNode := parsed[len(parsed)-1]
	exprNode.Ancestor = parsed[:len(parsed)-1]
	return &exprNode, nil
}

func splitAttrs(source string) []string {
	var parts []string
	current := strings.Builder{}
	inQuote := false
	quote := byte(0)
	for i := 0; i < len(source); i++ {
		c := source[i]
		if inQuote {
			current.WriteByte(c)
			if c == quote {
				inQuote = false
			}
			continue
		}
		switch c {
		case '\'', '"':
			inQuote = true
			quote = c
			current.WriteByte(c)
		case ' ':
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

func trimQuotes(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && (value[0] == '\'' || value[0] == '"') && value[len(value)-1] == value[0] {
		return value[1 : len(value)-1]
	}
	return value
}

func matches(node *Node, expr *XPathExpr) bool {
	if expr.Tag != "" && !strings.EqualFold(node.Name, expr.Tag) {
		return false
	}
	for key, value := range expr.Attrs {
		if node.Attr(key) != value {
			return false
		}
	}
	return true
}

func Find(root *Node, expr string) (*Node, error) {
	parsed, err := ParseXPath(expr)
	if err != nil {
		return nil, err
	}
	ancestors := parsed.Ancestor
	var match func(node *Node, depth int) *Node
	match = func(node *Node, depth int) *Node {
		if depth < len(ancestors) {
			if !matches(node, &ancestors[depth]) {
				return nil
			}
			for _, child := range node.Children {
				if found := match(child, depth+1); found != nil {
					return found
				}
			}
			return nil
		}
		if matches(node, parsed) {
			return node
		}
		for _, child := range node.Children {
			if found := match(child, depth); found != nil {
				return found
			}
		}
		return nil
	}

	if len(ancestors) > 0 {
		if !matches(root, &ancestors[0]) {
			for _, child := range root.Children {
				if found := match(child, 0); found != nil {
					return found, nil
				}
			}
			return nil, nil
		}
	}
	return match(root, 0), nil
}
