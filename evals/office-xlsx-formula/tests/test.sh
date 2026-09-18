test -f report.xlsx || exit 1
unzip -p report.xlsx xl/worksheets/sheet1.xml 2>/dev/null | grep -q SUM || exit 1
