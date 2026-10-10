test ! -f follow-isolated.txt
test -f follow-ok.txt && grep -q ok follow-ok.txt
