if exists("b:current_syntax")
  finish
endif

let b:current_syntax = ""
unlet b:current_syntax
runtime! syntax/html.vim
unlet b:current_syntax

let b:current_syntax = ""
unlet b:current_syntax
syn include @CSS syntax/css.vim
unlet b:current_syntax

let b:current_syntax = ""
unlet b:current_syntax
syn include @JS syntax/javascript.vim
unlet b:current_syntax

syn region gxcFrontmatter start=/\%^---$/ end=/^---$/ contains=gxcGoKeyword,gxcGoType,gxcGoString,gxcGoNumber,gxcGoComment,gxcGoBoolean
syn keyword gxcGoKeyword var import package type struct func if else for range return const contained
syn keyword gxcGoType string int int64 float64 bool byte rune contained
syn keyword gxcGoBoolean true false nil contained
syn match gxcGoType /\[\]\w\+/ contained
syn match gxcGoType /map\[\w\+\]\w\+/ contained
syn region gxcGoString start=/"/ skip=/\\"/ end=/"/ contained
syn region gxcGoString start=/`/ end=/`/ contained
syn match gxcGoNumber /\b\d\+\(\.\d\+\)\?/ contained
syn match gxcGoComment /\/\/.*$/ contained

syn region gxcStyle matchgroup=htmlTag start=+<style[^>]*>+ end=+</style>+ contains=@CSS keepend
syn region gxcScript matchgroup=htmlTag start=+<script[^>]*>+ end=+</script>+ contains=@JS keepend

syn region gxcInterpolation start=/{/ end=/}/ contained containedin=htmlString,htmlValue,htmlH1,htmlH2,htmlH3,htmlH4,htmlH5,htmlH6,htmlHead,htmlTitle,htmlBoldItalicUnderline,htmlUnderlineBold,htmlUnderlineItalicBold,htmlUnderlineBoldItalic,htmlItalicBold,htmlItalicBoldUnderline,htmlItalicUnderlineBold,htmlBoldUnderlineItalic,htmlBoldItalic,htmlBoldUnderline,htmlItalicUnderline

syn match gxcDirective /galaxy:\(if\|for\|else\|elsif\)/ contained

hi def link gxcGoKeyword Keyword
hi def link gxcGoType Type
hi def link gxcGoBoolean Boolean
hi def link gxcGoString String
hi def link gxcGoNumber Number
hi def link gxcGoComment Comment
hi def link gxcDirective Special
hi def link gxcInterpolation Identifier

let b:current_syntax = "gxc"
