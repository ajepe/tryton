package xml

import (
	"fmt"
	"sort"
	"strings"
)

type View struct {
	ID       int
	Arch     string
	Inherit  int
	Priority int
}

func ApplyInheritance(root *Node, extension *Node) error {
	directives := collectDirectives(extension)
	for _, directive := range directives {
		position := directive.Attr("position")
		if position == "" {
			continue
		}
		expr := directive.Attr("expr")
		if expr == "" {
			expr = exprFromNode(directive)
		}
		if expr == "" {
			return fmt.Errorf("inheritance directive without expr")
		}
		target, parent, err := findWithParent(root, expr)
		if err != nil {
			return err
		}
		if target == nil {
			return fmt.Errorf("xpath %q not found", expr)
		}

		switch position {
		case "replace":
			replacements := cloneChildren(directive)
			if len(replacements) != 1 {
				return fmt.Errorf("replace on %q requires exactly one child", expr)
			}
			if parent == nil {
				*root = *replacements[0]
				continue
			}
			index := childIndex(parent, target)
			parent.Children[index] = replacements[0]
		case "after":
			if parent == nil {
				return fmt.Errorf("cannot insert after root")
			}
			index := childIndex(parent, target)
			insertChildren(parent, index+1, cloneChildren(directive))
		case "before":
			if parent == nil {
				return fmt.Errorf("cannot insert before root")
			}
			index := childIndex(parent, target)
			insertChildren(parent, index, cloneChildren(directive))
		case "inside":
			target.Children = append(target.Children, cloneChildren(directive)...)
		case "attributes":
			applyAttributes(target, directive)
		default:
			return fmt.Errorf("unsupported position %q", position)
		}
	}
	return nil
}

func collectDirectives(extension *Node) []*Node {
	directives := []*Node{}
	if extension.Attr("position") != "" {
		directives = append(directives, extension)
	}
	for _, child := range extension.Children {
		if child.Attr("position") != "" {
			directives = append(directives, child)
			continue
		}
		directives = append(directives, collectDirectives(child)...)
	}
	return directives
}

func exprFromNode(node *Node) string {
	if node.Name == "" {
		return ""
	}
	if name := node.Attr("name"); name != "" {
		return fmt.Sprintf("//%s[@name='%s']", node.Name, name)
	}
	return "//" + node.Name
}

func cloneChildren(node *Node) []*Node {
	children := make([]*Node, 0, len(node.Children))
	for _, child := range node.Children {
		children = append(children, child.Clone())
	}
	return children
}

func applyAttributes(target *Node, directive *Node) {
	for _, attribute := range directive.Children {
		if attribute.Name != "attribute" {
			continue
		}
		name := attribute.Attr("name")
		if name == "" {
			continue
		}
		value := strings.TrimSpace(attribute.Text)
		if value == "" {
			delete(target.Attrs, name)
		} else {
			target.SetAttr(name, value)
		}
	}
}

func childIndex(parent, child *Node) int {
	for index, item := range parent.Children {
		if item == child {
			return index
		}
	}
	return -1
}

func insertChildren(parent *Node, index int, children []*Node) {
	if index < 0 {
		index = len(parent.Children)
	}
	tail := append([]*Node{}, parent.Children[index:]...)
	parent.Children = append(parent.Children[:index], children...)
	parent.Children = append(parent.Children, tail...)
}

func findWithParent(root *Node, expr string) (*Node, *Node, error) {
	parsed, err := ParseXPath(expr)
	if err != nil {
		return nil, nil, err
	}
	var match func(node *Node, depth int) *Node
	match = func(node *Node, depth int) *Node {
		ancestors := parsed.Ancestor
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

	target := match(root, 0)
	if target == nil {
		return nil, nil, nil
	}
	if target == root {
		return target, nil, nil
	}
	return target, locateParent(root, target), nil
}

func locateParent(node, target *Node) *Node {
	for _, child := range node.Children {
		if child == target {
			return node
		}
		if parent := locateParent(child, target); parent != nil {
			return parent
		}
	}
	return nil
}

func MergeViews(views []View) (string, error) {
	if len(views) == 0 {
		return "", fmt.Errorf("no views to merge")
	}
	sorted, err := sortViews(views)
	if err != nil {
		return "", err
	}
	base, err := ParseArch(sorted[0].Arch)
	if err != nil {
		return "", err
	}
	for _, view := range sorted[1:] {
		extension, err := ParseArch(view.Arch)
		if err != nil {
			return "", err
		}
		if err := ApplyInheritance(base, extension); err != nil {
			return "", fmt.Errorf("view %d: %w", view.ID, err)
		}
	}
	return MarshalArch(base)
}

func sortViews(views []View) ([]View, error) {
	byID := map[int]View{}
	for _, view := range views {
		byID[view.ID] = view
	}
	depth := map[int]int{}
	var compute func(id int) int
	compute = func(id int) int {
		if value, ok := depth[id]; ok {
			return value
		}
		view := byID[id]
		if view.Inherit == 0 {
			depth[id] = 0
			return 0
		}
		if _, ok := byID[view.Inherit]; !ok {
			depth[id] = 1
			return 1
		}
		depth[id] = compute(view.Inherit) + 1
		return depth[id]
	}

	sorted := append([]View{}, views...)
	for _, view := range sorted {
		if view.Inherit != 0 {
			if _, ok := byID[view.Inherit]; !ok {
				return nil, fmt.Errorf("inherited view %d not found", view.Inherit)
			}
		}
		_ = compute(view.ID)
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		if depth[sorted[i].ID] != depth[sorted[j].ID] {
			return depth[sorted[i].ID] < depth[sorted[j].ID]
		}
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority < sorted[j].Priority
		}
		return sorted[i].ID < sorted[j].ID
	})
	return sorted, nil
}
