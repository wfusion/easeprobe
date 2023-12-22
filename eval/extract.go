/*
 * Copyright (c) 2022, MegaEase
 * All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package eval

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	hq "github.com/antchfx/htmlquery"
	jq "github.com/antchfx/jsonquery"
	xq "github.com/antchfx/xmlquery"
	"github.com/wfusion/gofusion/common/utils"
	"github.com/wfusion/gofusion/common/utils/serialize/json"
	"golang.org/x/net/html"
)

// Extractor is the interface for all extractors
type Extractor interface {
	SetQuery(string)
	SetVarType(VarType)
	SetDocument(string)
	Extract() (interface{}, error)
}

// BaseExtractor is the base extractor
type BaseExtractor struct {
	Name         string  `yaml:"name"` // variable name
	VarType      VarType `yaml:"type"` // variable type
	Document     string  `yaml:"-"`
	DocType      DocType `yaml:"-"`
	XPath        string  `yaml:"-"`
	ExtractStrFn func() (string, error)
}

// SetVarType sets the variable type
func (x *BaseExtractor) SetVarType(t VarType) {
	x.VarType = t
}

// SetDocument sets the document
func (x *BaseExtractor) SetDocument(doc string) {
	x.Document = doc
}

// Extract extracts the value from the document by xpath expression
func (x *BaseExtractor) Extract() (interface{}, error) {
	switch x.VarType {
	case String:
		return x.ExtractStrFn()
	case Int:
		return x.ExtractInt()
	case Float:
		return x.ExtractFloat()
	case Bool:
		return x.ExtractBool()
	case Time:
		return x.ExtractTime()
	case Duration:
		return x.ExtractDuration()
	case LengthVariable:
		return x.ExtractLengthVariable()
	}
	return nil, fmt.Errorf("unknown type: %s", x.VarType)
}

// ExtractInt extracts the value from the document by xpath expression
func (x *BaseExtractor) ExtractInt() (int, error) {
	s, err := x.ExtractStrFn()
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(s)
}

// ExtractFloat extracts the value from the document by xpath expression
func (x *BaseExtractor) ExtractFloat() (float64, error) {
	s, err := x.ExtractStrFn()
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(s, 64)
}

// ExtractBool extracts the value from the document by xpath expression
func (x *BaseExtractor) ExtractBool() (bool, error) {
	s, err := x.ExtractStrFn()
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(s)
}

// ExtractTime extracts the value from the document by xpath expression
func (x *BaseExtractor) ExtractTime() (time.Time, error) {
	s, err := x.ExtractStrFn()
	if err != nil {
		return time.Time{}, err
	}
	return tryParseTime(s)
}

// copy from: https://github.com/Knetic/govaluate/blob/master/parsing.go#L473
func tryParseTime(str string) (time.Time, error) {

	timeFormats := [...]string{
		time.ANSIC,
		time.UnixDate,
		time.RubyDate,
		time.Kitchen,
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02",                         // RFC 3339
		"2006-01-02 15:04",                   // RFC 3339 with minutes
		"2006-01-02 15:04:05",                // RFC 3339 with seconds
		"2006-01-02 15:04:05-07:00",          // RFC 3339 with seconds and timezone
		"2006-01-02T15Z0700",                 // ISO8601 with hour
		"2006-01-02T15:04Z0700",              // ISO8601 with minutes
		"2006-01-02T15:04:05Z0700",           // ISO8601 with seconds
		"2006-01-02T15:04:05.999999999Z0700", // ISO8601 with nanoseconds
	}

	for _, format := range timeFormats {
		ret, err := tryParseExactTime(str, format)
		if err == nil {
			return ret, nil
		}
	}

	return time.Time{}, fmt.Errorf("Cannot parse the time: %s", str)
}

func tryParseExactTime(candidate string, format string) (time.Time, error) {
	var ret time.Time
	var err error

	ret, err = time.ParseInLocation(format, candidate, time.Local)
	if err != nil {
		return time.Time{}, err
	}
	return ret, nil
}

// ExtractDuration extracts the value from the document by xpath expression
func (x *BaseExtractor) ExtractDuration() (time.Duration, error) {
	s, err := x.ExtractStrFn()
	if err != nil {
		return 0, err
	}
	return time.ParseDuration(s)
}

func (x *BaseExtractor) ExtractLengthVariable() (any, error) {
	s, err := x.ExtractStrFn()
	if err != nil {
		return nil, err
	}
	switch x.DocType {
	case HTML:
		root, err := html.Parse(strings.NewReader(s))
		if err != nil {
			return nil, err
		}
		nodes, err := hq.QueryAll(root, x.XPath)
		if err != nil {
			return nil, err
		}
		values := make([]any, 0, len(nodes))
		for i := 0; i < len(nodes); i++ {
			for child := nodes[i].FirstChild; child != nil; child = child.NextSibling {
				switch child.Type {
				case html.ElementNode:
					for grandChild := child.FirstChild; grandChild != nil; grandChild = grandChild.NextSibling {
						if grandChild.Type == html.TextNode {
							values = append(values, hq.InnerText(grandChild))
						}
					}
				default:
					if child.Type == html.TextNode {
						values = append(values, hq.InnerText(child))
					}
				}
			}
		}
		return values, nil
	case XML:
		root, err := xq.Parse(strings.NewReader(s))
		if err != nil {
			return nil, err
		}
		node := root.SelectElement(x.XPath)
		var values []any
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			switch child.Type {
			case xq.ElementNode:
				for grandChild := child.FirstChild; grandChild != nil; grandChild = grandChild.NextSibling {
					values = append(values, grandChild.OutputXML(true))
				}
			default:
				if child.Type == xq.TextNode {
					values = append(values, child.Data)
				}
			}
		}
		return values, nil
	case TEXT, JSON:
		parent, err := jq.Parse(strings.NewReader(s))
		if err != nil {
			return nil, err
		}
		return utils.SliceMapping(parent.ChildNodes(), func(n *jq.Node) any { return n.Value() }), nil
	default:
		return nil, fmt.Errorf("Cannot parse the length: %s", s)
	}
}

// -----------------------------------------------------------------------------

// XPathNode is the generic type for xpath node
type XPathNode interface {
	jq.Node | xq.Node | html.Node
}

// XPathExtractor is a struct for extracting values from a html/xml/json string
type XPathExtractor[T XPathNode] struct {
	BaseExtractor
	XPath  string `yaml:"xpath"` // xpath expression
	Parser func(string) (*T, error)
	Query  func(*T, string) (*T, error)
	Inner  func(*T) string
}

// SetQuery sets the xpath expression
func (x *XPathExtractor[T]) SetQuery(q string) {
	x.XPath = q
	x.BaseExtractor.XPath = q
}

// Query query the string from the document by xpath expression
func Query[T XPathNode](document, xpath string,
	parser func(string) (*T, error),
	query func(*T, string) (*T, error),
	inner func(*T) string) (string, error) {
	doc, err := parser(document)
	if err != nil {
		return "", err
	}
	n, err := query(doc, xpath)
	if err != nil {
		return "", err
	}
	if n == nil {
		return "", nil
	}
	return inner(n), nil
}

// ExtractStr extracts the value from the document by xpath expression
func (x *XPathExtractor[T]) ExtractStr() (string, error) {
	return Query(x.Document, x.XPath, x.Parser, x.Query, x.Inner)
}

// NewJSONExtractor creates a new JSONExtractor
func NewJSONExtractor(document string, docType DocType) *XPathExtractor[jq.Node] {
	x := &XPathExtractor[jq.Node]{
		BaseExtractor: BaseExtractor{
			VarType:  String,
			Document: document,
			DocType:  docType,
		},
		Parser: func(document string) (*jq.Node, error) {
			return jq.Parse(strings.NewReader(document))
		},
		Query: func(doc *jq.Node, xpath string) (*jq.Node, error) {
			return jq.Query(doc, xpath)
		},
		Inner: func(n *jq.Node) string {
			switch n.Type {
			case jq.DocumentNode, jq.TextNode:
				return n.InnerText()
			default:
				switch n.Value().(type) {
				case []any, []map[string]any, map[string]any:
					bs, _ := json.Marshal(n.Value())
					return string(bs)
				default:
					return n.InnerText()
				}
			}
		},
	}
	x.ExtractStrFn = x.ExtractStr
	return x
}

// NewXMLExtractor creates a new XMLExtractor
func NewXMLExtractor(document string, docType DocType) *XPathExtractor[xq.Node] {
	x := &XPathExtractor[xq.Node]{
		BaseExtractor: BaseExtractor{
			VarType:  String,
			Document: document,
			DocType:  docType,
		},
		Parser: func(document string) (*xq.Node, error) {
			return xq.Parse(strings.NewReader(document))
		},
		Query: func(doc *xq.Node, xpath string) (*xq.Node, error) {
			return xq.Query(doc, xpath)
		},
		Inner: func(n *xq.Node) string {
			switch n.Type {
			case xq.DocumentNode,
				xq.TextNode,
				xq.DeclarationNode,
				xq.CharDataNode,
				xq.CommentNode,
				xq.AttributeNode:
				return n.InnerText()
			default:
				if n.FirstChild == nil || (n.FirstChild.Type == xq.TextNode && n.FirstChild.NextSibling == nil) {
					return n.InnerText()
				} else {
					return n.OutputXML(true)
				}
			}
		},
	}
	x.ExtractStrFn = x.ExtractStr
	return x
}

// NewHTMLExtractor creates a new HTMLExtractor
func NewHTMLExtractor(document string, docType DocType) *XPathExtractor[html.Node] {
	x := &XPathExtractor[html.Node]{
		BaseExtractor: BaseExtractor{
			VarType:  String,
			Document: document,
			DocType:  docType,
		},
		Parser: func(document string) (*html.Node, error) {
			return html.Parse(strings.NewReader(document))
		},
		Query: func(doc *html.Node, xpath string) (*html.Node, error) {
			nodes, err := hq.QueryAll(doc, xpath)
			if err != nil {
				return nil, err
			}
			if len(nodes) == 1 {
				return nodes[0], nil
			}
			for i := 0; i < len(nodes); i++ {
				nodes[i].PrevSibling = nil
				nodes[i].NextSibling = nil
			}
			root := &html.Node{
				Type:       html.ElementNode,
				FirstChild: nodes[0],
				LastChild:  nodes[len(nodes)-1],
			}
			for i, child := 1, root.FirstChild; i < len(nodes); i++ {
				child.NextSibling = nodes[i]
				child.NextSibling.PrevSibling = child
				child.PrevSibling = nodes[i-1]
				child = child.NextSibling
			}
			return root, nil
		},
		Inner: func(n *html.Node) string {
			switch n.Type {
			case html.DocumentNode,
				html.TextNode,
				html.ErrorNode,
				html.CommentNode,
				html.DoctypeNode:
				return hq.InnerText(n)
			default:
				if n.FirstChild == nil || (n.FirstChild.Type == html.TextNode && n.FirstChild.NextSibling == nil) {
					return hq.InnerText(n)
				} else {
					return hq.OutputHTML(n, false)
				}
			}
		},
	}
	x.Document = document
	x.ExtractStrFn = x.ExtractStr
	return x
}

//------------------------------------------------------------------------------

// RegexExtractor is a struct for extracting values from a plain string
type RegexExtractor struct {
	BaseExtractor
	Regex string `yaml:"regex"` // regex expression
}

// SetQuery sets the regex expression
func (r *RegexExtractor) SetQuery(q string) {
	r.Regex = q
}

// MatchStr matches the string with the regex expression
func (r *RegexExtractor) MatchStr() (string, error) {
	re := regexp.MustCompile(r.Regex)
	match := re.FindStringSubmatch(r.Document)
	if match == nil {
		return "", fmt.Errorf("no match found for - %s", r.Regex)
	}
	for i, name := range re.SubexpNames() {
		if i > 0 && i <= len(match) {
			if len(name) > 0 {
				r.Name = name
			}
			return match[i], nil
		}
	}
	return match[0], nil
}

// NewRegexExtractor creates a new RegexExtractor
func NewRegexExtractor(document string, docType DocType) *RegexExtractor {
	x := &RegexExtractor{
		BaseExtractor: BaseExtractor{
			VarType:  String,
			Document: document,
			DocType:  docType,
		},
	}
	x.ExtractStrFn = x.MatchStr
	return x
}
