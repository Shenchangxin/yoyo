test ! -f role-in-harbor.txt
test -f harbor-ok.txt && grep -q ok harbor-ok.txt
