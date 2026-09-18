test -f deck.pptx || exit 1
n=$(unzip -l deck.pptx 2>/dev/null | grep -c 'ppt/slides/slide.*xml' || true)
test "$n" -ge 3
