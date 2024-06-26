package main

import (
	"embed"
	"log"
	"path/filepath"
	"strings"
)

var (
	//go:embed assets/*.js
	js embed.FS
	//go:embed assets/*.css
	css embed.FS

	jsStr         string
	mathjaxCfgStr string
	mathjaxStr    string
	cssStr        string
)

func init() {
	jsStr = getEmbedFilesToStr(js, "assets", "mathjax", true)
	mathjaxCfgStr = getEmbedFilesToStr(js, "assets", "mathjax-config.min.js", false)
	mathjaxStr = getEmbedFilesToStr(js, "assets", "MathJax-TeXSVG.min.js", false)
	cssStr = getEmbedFilesToStr(css, "assets", "", false)
}

func getEmbedFilesToStr(embedFS embed.FS, embedDir, keyword string, exclude bool) string {
	resultBuilder := strings.Builder{}
	if embedDirs, err := embedFS.ReadDir(embedDir); err == nil {
		var onlyThis bool
		for _, dir := range embedDirs {
			currName := dir.Name()
			embedPath := strings.Join([]string{embedDir, currName}, "/")
			if dir.IsDir() {
				resultBuilder.WriteString(getEmbedFilesToStr(embedFS, embedPath, keyword, exclude))
				continue
			}
			lowerName := strings.ToLower(currName)
			if keyword != "" {
				found := strings.Contains(lowerName, strings.ToLower(keyword))
				if exclude && found {
					continue // 排除
				}
				onlyThis = !exclude && found // 只获取匹配的文件内容
			}

			// 读取文件内容为字符串
			var embedContent []byte
			if embedContent, err = embedFS.ReadFile(embedPath); err != nil {
				log.SetPrefix("[WARN]")
				log.Println("Read embed file content error: ", err)
				continue
			}

			var prefix, suffix string
			switch filepath.Ext(lowerName) {
			case ".js":
				name := filepath.Base(lowerName)
				if strings.Contains(name, "mathjax") && strings.Contains(name, "config") {
					prefix = "<script type=\"text/x-mathjax-config\">"
				} else {
					prefix = "<script type=\"text/javascript\">"
				}
				prefix += " /* " + currName + " */\n"
				suffix = "\n</script>\n"
			case ".css":
				prefix = "<style> /* " + currName + " */\n"
				suffix = "\n</style>\n"
			default:
			}

			if onlyThis {
				resultBuilder.Reset()
			}
			resultBuilder.WriteString(prefix)
			resultBuilder.Write(embedContent)
			resultBuilder.WriteString(suffix)

			if onlyThis {
				break
			}
		}
	} else {
		log.SetPrefix("[WARN]")
		log.Println("Read embed file error: ", err)
	}
	return resultBuilder.String()
}
