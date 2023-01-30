@echo off
color 07
title 生成 GitHub Pages 页面
:: file-encoding=GBK
rem by iTanken

cd /d %~dp0/../../
go run . example/*.md -e -t -m -s -f example/img/go.png -c example/css/custom-css.css -o gh-pages/index.html --html-title 示例文档

call "%~dp0/done-time-pause.bat"
