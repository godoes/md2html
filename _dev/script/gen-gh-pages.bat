@echo off
color 07
title ���� GitHub Pages ҳ��
:: file-encoding=GBK
rem by iTanken

cd /d %~dp0/../../
go run . example/*.md -e -t -m -s -f example/img/go.png -c example/css/custom-css.css -o gh-pages/index.html --html-title ʾ���ĵ�

call "%~dp0/done-time-pause.bat"
