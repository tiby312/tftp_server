atftp -p -l test2.png localhost $1 &&

atftp -g -l test_out.png -r test2.png localhost $1
