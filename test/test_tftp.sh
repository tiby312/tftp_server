atftp -p -l test.png localhost $1 &&

atftp -g -l test_out.png -r test.png localhost $1
