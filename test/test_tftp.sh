atftp -p -l $2 localhost $1 &&

atftp -g -l out.zip -r $2 localhost $1
