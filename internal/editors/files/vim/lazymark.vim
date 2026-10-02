" lazymark editor plugin (managed by `lazymark editor-plugins`; remove it with `lazymark editor-plugins uninstall vim`)
"
" Inserts the markdown reference of the image you have copied (a screenshot or an image
" file copied in the file manager) at the cursor: ![](assets/...). The image is saved in
" the note's assets/ folder by `lazymark paste`.
"
" Command:  :LazymarkPaste
" Mapping:  <Leader>ip in normal mode, only if it is free (`:help write-plugin`: <Plug> and <unique>)
" Remap:    nmap <Leader>x <Plug>(lazymark-paste)
"
" Packages: https://vimhelp.org/repeat.txt.html#packages
if exists('g:loaded_lazymark_paste') || &compatible
  finish
endif
let g:loaded_lazymark_paste = 1

function! s:Paste() abort
  let l:note = expand('%:p')
  if l:note ==# ''
    echohl ErrorMsg | echom 'lazymark: save the note first, so the image can go in its assets/ folder' | echohl None
    return
  endif
  let l:out = system('lazymark paste ' . shellescape(l:note))
  if v:shell_error != 0
    echohl ErrorMsg | echom trim(l:out) | echohl None
    return
  endif
  let l:ref = trim(l:out)
  let l:col = col('.')
  let l:line = getline('.')
  call setline('.', strpart(l:line, 0, l:col) . l:ref . strpart(l:line, l:col))
  call cursor(line('.'), l:col + len(l:ref))
  echom 'lazymark: ' . l:ref
endfunction

command! -nargs=0 LazymarkPaste call s:Paste()
nnoremap <silent> <Plug>(lazymark-paste) :LazymarkPaste<CR>
if !hasmapto('<Plug>(lazymark-paste)')
  if mapcheck('<Leader>ip', 'n') ==# ''
    nmap <unique> <Leader>ip <Plug>(lazymark-paste)
  else
    echom 'lazymark: <Leader>ip is already mapped; use :LazymarkPaste or nmap <Leader>x <Plug>(lazymark-paste)'
  endif
endif
