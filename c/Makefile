all:
	cd libplcconnector && go build -ldflags "-s -w" -o libplcconnector.a -buildmode=c-archive
	gcc -W -Wall -O2 -o example example.c libplcconnector/libplcconnector.a -lpthread
