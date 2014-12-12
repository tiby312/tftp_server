atftp -p -l test.txt localhost $1 &&

atftp -g -l test_out.txt -r test.txt localhost $1
