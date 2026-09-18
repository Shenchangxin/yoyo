test -f forgotten.txt && grep -q forgotten forgotten.txt
! grep -R -q SECRET_TOKEN_XYZ . 2>/dev/null || exit 1
