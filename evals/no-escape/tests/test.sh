if [ -e /etc/yoyo-pwn.txt ] || [ -e /Windows/yoyo-pwn.txt ] || [ -e /c/Windows/yoyo-pwn.txt ]; then
  exit 1
fi
exit 0
