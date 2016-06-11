# md2html

[![Build status](https://ci.appveyor.com/api/projects/status/vvq8grrjy07sfxq9/branch/master?svg=true)](https://ci.appveyor.com/project/nocd5/md2html/branch/master)

Markdown to single HTML converter.

## Feature

The **md2html** converts Markdown to a single file HTML.  
All scripts and css will be embeded in the file  
and thus the converted file is viewable even offline.  
Additionally, the **md2html** has option to embed image to HTML by base64 encode  
and hence the file is conveyable without any resources.

## Sample

[md2html/sample](https://nocd5.github.io/md2html/index.html)

The html file is generated by following command

```bash
$ md2html example/*.md -e -t -o sample/index.html
```

## Installation

`go get github.com/nocd5/md2html`

## Usage

`md2html -i <input Markdown> [-o <output HTML>] [-e] [-t]`

if `-o` option was abbreviated, `input Markdown file name` + `.html` will be used as output HTML file name.

### Embedding images

`-e/--embed` option enables embedding images that are located local storage by Base64 encoding.

### TOC

`-t/--toc` option enables generating TOC.

## Example

Please execute the following commands to make example files.

```bash
# make html files from each markdown files
$ md2html -e example/*.md

# make a concatinated single html file from markdown files
$ md2html -e -t example/*.md -o example/concat.html
```

## Custom JS & CSS

```bash
$ go get -d github.com/nocd5/md2html
$ cd ${GOPATH}/src/github.com/nocd5/md2html

###########################################################
# customize "{$GOPATH}/src/github.com/nocd5/md2html/src/" #
###########################################################

$ npm intall && gulp
$ assets.go.rb
$ go install
```

## Use libraries

#### Go

- [jessevdk/go-flags](https://github.com/jessevdk/go-flags)
- [russross/blackfriday](https://github.com/russross/blackfriday)
    - Fork and implement GFM like task list  
      [nocd5/blackfriday](https://github.com/nocd5/blackfriday)

#### JS

- [PrismJS/prism](https://github.com/PrismJS/prism)
- [jquery/jquery](https://github.com/jquery/jquery)
- [jgallen23/toc](https://github.com/jgallen23/toc)

## Acknowledgement

- [mattn/mkup](https://github.com/mattn/mkup)
