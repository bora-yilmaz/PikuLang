package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Token structure
type Token struct {
	Type  string
	Value string
}

// Node structure representing an element or list
type Node struct {
	Type     string
	Value    string
	Children []*Node
}

// Tokenize the input string
func tokenize(source string) ([]Token, error) {
	tokenSpec := []struct {
		pattern string
		typeStr string
	}{
		{`^\d+`, "INTEGER"},
		{`^[a-zA-Z_][a-zA-Z_0-9]*`, "IDENTIFIER"},
		{`^\[`, "LBRACKET"},
		{`^\]`, "RBRACKET"},
		{`^\s+`, "WHITESPACE"},
	}

	var tokens []Token
	source = strings.TrimSpace(source)
	for len(source) > 0 {
		matched := false
		for _, spec := range tokenSpec {
			re := regexp.MustCompile(spec.pattern)
			match := re.FindString(source)
			if match != "" {
				if spec.typeStr != "WHITESPACE" {
					tokens = append(tokens, Token{Type: spec.typeStr, Value: match})
				}
				source = source[len(match):]
				matched = true
				break
			}
		}
		if !matched {
			return nil, fmt.Errorf("unexpected character: %v", source)
		}
	}
	return tokens, nil
}

// Parse a list
func parseList(tokens []Token) (*Node, []Token, error) {
	if len(tokens) == 0 || tokens[0].Type != "LBRACKET" {
		return nil, tokens, fmt.Errorf("expected '[' at the beginning of the list")
	}

	tokens = tokens[1:]
	rootNode := &Node{Type: "LIST", Children: []*Node{}}

	for len(tokens) > 0 && tokens[0].Type != "RBRACKET" {
		token := tokens[0]
		if token.Type == "INTEGER" || token.Type == "IDENTIFIER" {
			node := &Node{Type: token.Type, Value: token.Value}
			rootNode.Children = append(rootNode.Children, node)
			tokens = tokens[1:]
		} else if token.Type == "LBRACKET" {
			nestedNode, remainingTokens, err := parseList(tokens)
			if err != nil {
				return nil, nil, err
			}
			rootNode.Children = append(rootNode.Children, nestedNode)
			tokens = remainingTokens
		} else {
			return nil, nil, fmt.Errorf("unexpected token: %v", token)
		}
	}

	if len(tokens) == 0 || tokens[0].Type != "RBRACKET" {
		return nil, nil, fmt.Errorf("expected ']' at the end of the list")
	}

	tokens = tokens[1:]
	return rootNode, tokens, nil
}

// Parse multiple lists
func parseMultipleLists(tokens []Token) ([]*Node, error) {
	var nodes []*Node
	var err error

	for len(tokens) > 0 {
		if tokens[0].Type == "LBRACKET" {
			var node *Node
			node, tokens, err = parseList(tokens)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		} else {
			return nil, fmt.Errorf("unexpected token outside brackets: %v", tokens[0])
		}
	}

	return nodes, nil
}

type St struct {
	valt    string
	funcval *Function
	varval  int
}

type Env struct {
	vals map[string]*St
}

type Function struct {
	Args []string
	expr *Node
}

func eval(node *Node, env *Env, ln int) (*St, error, *Env) {
	switch node.Type {
	case "IDENTIFIER":
		v, ok := env.vals[node.Value]
		if ok {
			return v, nil, env
		}
		return nil, fmt.Errorf("undefined identifier: %s, line: %d", node.Value, ln), env
	case "INTEGER":
		a, err := strconv.Atoi(node.Value)
		if err == nil {
			return &St{valt: "n", varval: a}, nil, env
		}
		return nil, err, nil
	case "LIST":
		switch node.Children[0].Value {
		case "call":
			f, err, env := eval(node.Children[1], env, ln)
			if err != nil {
				return nil, err, nil
			}
			if f.valt == "f" {
				a, b, env := callfunc(f, env, ln, node.Children[2:])
				return a, b, env
			}
			return nil, fmt.Errorf("not a function: %s line: %d", node.Children[1].Value, ln), nil
		case "set":
			a, err, env := eval(node.Children[2], env, ln)
			if err == nil {
				env.vals[node.Children[1].Value] = a
				return nil, nil, env
			}
			return nil, err, nil
		case "echo":
			b, err2, env := eval(node.Children[1], env, ln)

			if err2 != nil {
				return nil, err2, nil
			}

			if b.valt == "n" {
				_, err := fmt.Println(b.varval)
				return nil, err, env
			}
			_, err := fmt.Printf("Unprintable Value: %s line: %d", node.Children[1].Value, ln)

			return nil, err, env
		case "func":
			arg := []string{}
			for _, a := range node.Children[1].Children {
				arg = append(arg, a.Value)
			}
			return &St{valt: "f", funcval: &Function{Args: arg, expr: node.Children[2]}}, nil, env
		case "add":
			av, err1, env := eval(node.Children[1], env, ln)
			if err1 != nil {
				return nil, err1, nil
			}
			a := av.varval
			bv, err2, env := eval(node.Children[2], env, ln)
			if err2 != nil {
				return nil, err2, nil
			}
			b := bv.varval

			return &St{valt: "n", varval: a + b}, nil, env
		case "sub":
			av, err1, env := eval(node.Children[1], env, ln)
			if err1 != nil {
				return nil, err1, env
			}
			a := av.varval
			bv, err2, env := eval(node.Children[2], env, ln)
			if err2 != nil {
				return nil, err2, nil
			}
			b := bv.varval

			return &St{valt: "n", varval: a - b}, nil, env
		case "mul":
			av, err1, env := eval(node.Children[1], env, ln)
			if err1 != nil {
				return nil, err1, nil
			}
			a := av.varval
			bv, err2, env := eval(node.Children[2], env, ln)
			if err2 != nil {
				return nil, err2, nil
			}
			b := bv.varval

			return &St{valt: "n", varval: a * b}, nil, env
		case "div":
			av, err1, env := eval(node.Children[1], env, ln)
			if err1 != nil {
				return nil, err1, nil
			}
			a := av.varval
			bv, err2, env := eval(node.Children[2], env, ln)
			if err2 != nil {
				return nil, err2, nil
			}
			b := bv.varval

			return &St{valt: "n", varval: a / b}, nil, env
		case "mod":
			av, err1, env := eval(node.Children[1], env, ln)
			if err1 != nil {
				return nil, err1, nil
			}
			a := av.varval
			bv, err2, env := eval(node.Children[2], env, ln)
			if err2 != nil {
				return nil, err2, nil
			}
			b := bv.varval

			return &St{valt: "n", varval: a % b}, nil, env
		case "neg":
			av, err1, env := eval(node.Children[1], env, ln)
			if err1 != nil {
				return nil, err1, env
			}
			a := av.varval

			return &St{valt: "n", varval: 0-a}, nil, env
		case "import":
			env, err := runfile(node.Children[1].Value+".pi", env)
			if err != nil {
				return nil, err, nil
			}
			return nil, nil, env
		case "if":
			res, err, env := eval(node.Children[1], env, ln)
			if err != nil{
				return nil, err, nil
			}
			if res.valt == "n" && res.varval <= 0{
				return eval(node.Children[3], env, ln)
			}
			return eval(node.Children[2], env, ln)
		default:
			return nil, fmt.Errorf("unknown command: %s, line: %d", node.Children[0].Value, ln), nil
		}
	default:
		return nil, fmt.Errorf("compiler internal error 181, line: %d", ln), nil
	}
}

func pass(a any) {
}

func callfunc(f *St, env *Env, ln int, args []*Node) (*St, error, *Env) {
	for i, a := range f.funcval.Args {
		x, err, env := eval(args[i], env, ln)
		if err != nil {
			return nil, err, nil
		}
		env.vals[a] = x
	}
	//scope generated at this point
	return eval(f.funcval.expr, env, ln)
}

func execast(nodes []*Node, env *Env) (*Env, error) {
	for i, node := range nodes {
		_, err, nenv := eval(node, env, i)
		env = nenv
		if err != nil {
			return nil, err
		}
	}
	return env, nil
}

func LoadFile(filename string) ([]*Node, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	source := string(data)
	tokens, err := tokenize(source)
	if err != nil {
		return nil, err
	}
	return parseMultipleLists(tokens)
}

func runfile(filename string, env *Env) (*Env, error) {
	code, err := LoadFile(filename)

	if err != nil {
		return nil, err
	}

	env, err2 := execast(code, env)

	if err2 != nil {
		return nil, err2
	}

	return env, nil

}

func main() {
	e := make(map[string]*St)
	_, err := runfile(os.Args[1], &Env{vals: e})
	if err != nil{
		fmt.Println("Error", err)
	}
}
