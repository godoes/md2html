package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/PuerkitoBio/goquery"
	chromaHtml "github.com/alecthomas/chroma/formatters/html"
	"github.com/jessevdk/go-flags"
	"github.com/nocd5/goldmark-highlighting"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
)

type Options struct {
	InputFile  string `long:"input" short:"i" description:"input Markdown"`
	OutputFile string `long:"output" short:"o" description:"output HTML"`
	EmbedImage bool   `long:"embed" short:"e" description:"embed image by base64 encoding"`
	TOC        bool   `long:"toc" short:"t" description:"generate TOC"`
	MathJax    bool   `long:"mathjax" short:"m" description:"use MathJax"`
	Favicon    string `long:"favicon" short:"f" description:"use favicon"`
	TableSpan  bool   `long:"span" short:"s" description:"enable table row/col span"`
	CustomCSS  string `long:"css" short:"c" description:"add custom CSS"`
}

const (
	template = `<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta http-equiv="X-UA-Compatible" content="IE=edge">
%s<title>%s</title>
%s
</head>
<body>
<div class="container">
%s<div class="markdown-body">
%s
</div>
</div>
</body>
</html>`

	tocTag = `<div id="markdown-toc"></div>
`
	faviconTag = `<link rel='shortcut icon' href='data:image/x-icon;base64,%s'/>
`
)

// goldmark convert options
var (
	extensions = []goldmark.Extender{
		extension.GFM,
		extension.DefinitionList,
		extension.Footnote,
		extension.Typographer,
		highlighting.NewHighlighting(
			highlighting.WithStyle("github"),
			highlighting.WithFormatOptions(
				chromaHtml.WithClasses(true),
			),
		),
	}
	parserOptions = []parser.Option{
		parser.WithAutoHeadingID(),
	}
	rendererOptions = []renderer.Option{
		html.WithXHTML(),
		html.WithUnsafe(),
	}
)

func main() {
	var opts Options
	inputs, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}

	if len(opts.InputFile) > 0 {
		inputs = []string{opts.InputFile}
	}

	if len(inputs) <= 0 {
		_, _ = fmt.Fprintln(os.Stderr, "Please specify input Markdown")
		os.Exit(1)
	}

	var files []string
	for _, input := range inputs {
		f, err := filepath.Glob(input)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		files = append(files, f...)
	}
	if len(files) <= 0 {
		_, _ = fmt.Fprintln(os.Stderr, "File is not found")
		os.Exit(1)
	}

	mdParser := goldmark.New(
		goldmark.WithExtensions(extensions...),
		goldmark.WithParserOptions(parserOptions...),
		goldmark.WithRendererOptions(rendererOptions...),
	)

	if len(opts.OutputFile) > 0 {
		ext := regexp.QuoteMeta(filepath.Ext(opts.OutputFile))
		re := regexp.MustCompile(ext + "$")
		title := filepath.Base(re.ReplaceAllString(opts.OutputFile, ""))
		htmlStr, err := renderHTMLConcat(files, opts.EmbedImage, mdParser)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := writeHTML(htmlStr, title, opts.OutputFile, opts.TOC, opts.MathJax, opts.Favicon, opts.TableSpan, opts.CustomCSS); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}
	} else {
		for _, file := range files {
			htmlStr, err := renderHTML(file, opts.EmbedImage, mdParser)
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			if err := writeHTML(htmlStr, file, file+".html", opts.TOC, opts.MathJax, opts.Favicon, opts.TableSpan, opts.CustomCSS); err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err)
			}
		}
	}
}

func renderHTML(input string, embed bool, parser goldmark.Markdown) (string, error) {
	fi, err := os.Open(input)
	if err != nil {
		return "", err
	}
	defer func(fi *os.File) {
		_ = fi.Close()
	}(fi)

	md, err := io.ReadAll(fi)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := parser.Convert(md, &buf); err != nil {
		return "", err
	}

	htmlStr, err := parseImageOpt(buf.String())
	if err != nil {
		return "", err
	}

	if embed {
		htmlStr, err = embedImage(htmlStr, filepath.Dir(input))
		if err != nil {
			return "", err
		}
	}

	return htmlStr, nil
}

func renderHTMLConcat(inputs []string, embed bool, parser goldmark.Markdown) (string, error) {
	htmlStr := ""
	for _, input := range inputs {
		fi, err := os.Open(input)
		if err != nil {
			return "", err
		}

		md, err := io.ReadAll(fi)
		if err != nil {
			_ = fi.Close()
			return "", err
		}

		var buf bytes.Buffer
		if err := parser.Convert(md, &buf); err != nil {
			_ = fi.Close()
			return "", err
		}

		h, err := parseImageOpt(buf.String())
		if err != nil {
			_ = fi.Close()
			return "", err
		}

		if embed {
			h, err = embedImage(h, filepath.Dir(input))
			if err != nil {
				_ = fi.Close()
				return "", err
			}
		}

		htmlStr += h
		_ = fi.Close()
	}

	return htmlStr, nil
}

func writeHTML(html, title, output string, toc, mathjax bool, favicon string, tableSpan bool, customCSS string) error {
	var err error

	js := string(jsBytes[:])
	if mathjax {
		js += string(mathjaxCfgBytes[:])
		js += string(mathjaxBytes[:])
	}

	css := string(cssBytes[:])

	tt := ""
	if toc {
		tt = tocTag
	}

	faviconEle := ""
	if len(favicon) > 0 {
		cwd, _ := os.Getwd()
		b, err := decodeBase64(favicon, cwd)
		if err != nil {
			return err
		}
		faviconEle = fmt.Sprintf(faviconTag, b)
	}

	if mathjax {
		html, err = replaceMathJaxCodeBlock(html)
		if err != nil {
			return err
		}
	}

	html, err = replaceCheckBox(html)
	if err != nil {
		return err
	}

	if tableSpan {
		html, err = replaceTableSpan(html)
		if err != nil {
			return err
		}
	}

	if len(customCSS) > 0 {
		fi, err := os.Open(customCSS)
		if err != nil {
			return err
		}
		defer func(fi *os.File) {
			_ = fi.Close()
		}(fi)

		c, err := io.ReadAll(fi)
		if err != nil {
			return err
		}
		css += "<style type=\"text/css\">\n" + string(c) + "</style>\n"
	}

	fo, err := os.Create(output)
	if err != nil {
		return err
	}
	defer func(fo *os.File) {
		_ = fo.Close()
	}(fo)

	_, _ = fmt.Fprintf(fo, template, faviconEle, title, js+"\n"+css, tt, html)
	return nil
}

func embedImage(src, parent string) (string, error) {
	dest := src

	reFind := regexp.MustCompile(`(<img[\S\s]+?src=")([\S\s]+?)("[\S\s]*?/?>)`)
	reUrl := regexp.MustCompile(`(?i)^https?://.*`)

	imgTags := reFind.FindAllString(src, -1)
	for _, t := range imgTags {
		imgSrc := reFind.ReplaceAllString(t, "$2")

		if reUrl.MatchString(imgSrc) {
			continue
		}
		b64img, err := decodeBase64(imgSrc, parent)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			continue
		}

		reReplace, err := regexp.Compile(`(<img[\S\s]+?src=")` + regexp.QuoteMeta(imgSrc) + `("[\S\s]*?/?>)`)
		if err != nil {
			return src, err
		}

		ext := filepath.Ext(imgSrc)
		mimeType := mime.TypeByExtension(ext)
		if len(mimeType) <= 0 {
			mimeType = "image"
		}
		dest = reReplace.ReplaceAllString(dest, "${1}data:"+mimeType+";base64,"+b64img+"${2}")
	}
	return dest, nil
}

func decodeBase64(src, parent string) (string, error) {
	path := src
	if !filepath.IsAbs(path) {
		path = filepath.Join(parent, path)
	}
	f, err := os.Open(path)
	if err != nil {
		pathErr := err.(*os.PathError)
		errno := pathErr.Err.(syscall.Errno)
		if errno != 0x7B { // suppress ERROR_INVALID_NAME
			_, _ = fmt.Fprintln(os.Stderr, err)
			return "", nil
		}
		return "", err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	d, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}

	dest := base64.StdEncoding.EncodeToString(d)
	return dest, nil
}

func parseImageOpt(src string) (string, error) {
	dest := src

	re := regexp.MustCompile(`(<img[\S\s]+?src=)"([\S\s]+?)\?(\S+?)"([\S\s]*?/?>)`)
	dest = re.ReplaceAllStringFunc(dest, func(s string) string {
		res := re.FindStringSubmatch(s)
		return res[1] + "\"" + res[2] + "\" " + strings.Join(strings.Split(res[3], "&amp;"), " ") + res[4]
	})
	return dest, nil
}

func replaceMathJaxCodeBlock(src string) (string, error) {
	sr := strings.NewReader(src)
	doc, err := goquery.NewDocumentFromReader(sr)
	if err != nil {
		return src, err
	}

	code := doc.Find("pre>code.language-math")
	code.Each(func(index int, s *goquery.Selection) {
		s.Parent().ReplaceWithHtml("<p>$$" + s.Text() + "$$</p>")
	})

	return doc.Find("body").Html()
}

func replaceCheckBox(src string) (string, error) {
	sr := strings.NewReader(src)
	doc, err := goquery.NewDocumentFromReader(sr)
	if err != nil {
		return src, err
	}

	doc.Find("li").Each(func(i int, li *goquery.Selection) {
		li.Contents().Each(func(j int, c *goquery.Selection) {
			if goquery.NodeName(c) == "#text" {
				li.Find("input").Each(func(k int, input *goquery.Selection) {
					if t, exist := input.Attr("type"); exist && t == "checkbox" {
						li.AddClass("task-list-item")
					}
				})
			}
		})
	})

	return doc.Find("body").Html()
}

func replaceTableSpan(src string) (string, error) {
	sr := strings.NewReader(src)
	doc, err := goquery.NewDocumentFromReader(sr)
	if err != nil {
		return src, err
	}

	re := regexp.MustCompile("\u00a6\\s*")

	doc.Find("table").Each(func(i int, tbl *goquery.Selection) {
		tbl.Find("tbody").Each(func(j int, tbody *goquery.Selection) {
			trs := tbody.Find("tr")
			// colspan
			colMax := 0
			trs.Each(func(k int, tr *goquery.Selection) {
				tds := tr.Find("td")
				colMns := tds.Length()
				if colMns > colMax {
					colMax = colMns
				}
				col := 0
				tds.Each(func(l int, td *goquery.Selection) {
					col++
					td.Contents().Each(func(m int, c *goquery.Selection) {
						cnt := len(re.FindAllIndex([]byte(c.Text()), -1))
						if cnt > 0 {
							td.SetAttr("colspan", strconv.Itoa(cnt+1))
							c.ReplaceWithHtml(re.ReplaceAllString(c.Text(), ""))
							col += cnt
						}
					})
					if col > colMns {
						td.SetAttr("hidden", "")
					}
				})
			})
			// rowspan
			for m := 0; m < colMax; m++ {
				var root *goquery.Selection
				cnt := 0
				trs.Each(func(k int, tr *goquery.Selection) {
					tr.Find("td").Each(func(l int, td *goquery.Selection) {
						if l == m {
							atd := getActualTD(tr, l)
							if k == 0 {
								root = atd
							} else {
								if atd.Text() != "" {
									cnt = 0
									root = atd
								} else {
									cnt++
									root.SetAttr("rowspan", strconv.Itoa(cnt+1))
									atd.SetAttr("hidden", "")
								}
							}
						}
					})
				})
			}
			// remove hidden <td>
			tbody.Find("tr>td").Each(func(i int, td *goquery.Selection) {
				if _, hidden := td.Attr("hidden"); hidden {
					td.Remove()
				}
			})
		})
		// remove empty header
		empty := true
		tbl.Find("thead").Each(func(i int, thead *goquery.Selection) {
			thead.Find("tr>th").EachWithBreak(func(j int, th *goquery.Selection) bool {
				if th.Text() != "" {
					empty = false
					return false
				}
				return true
			})
			if empty {
				thead.Remove()
			}
		})
	})

	return doc.Find("body").Html()
}

func getActualTD(tr *goquery.Selection, index int) *goquery.Selection {
	pos := 0
	var result *goquery.Selection
	tr.Find("td").EachWithBreak(func(i int, td *goquery.Selection) bool {
		cs, _ := strconv.Atoi(td.AttrOr("colspan", "1"))
		pos += cs
		if pos >= index+1 {
			result = td
			return false
		}
		return true
	})

	return result
}
