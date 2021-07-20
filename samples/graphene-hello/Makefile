all: sign
.PHONY: clean all


sign: hello.manifest hello premain-libos
	openssl genrsa -3 -out signer-key.pem 3072
	graphene-sgx-sign -output hello.manifest.sgx --manifest hello.manifest --key signer-key.pem
	graphene-sgx-get-token -sig hello.sig -output hello.token


clean:
	rm -f *.sig *.token *.manifest.sgx hello hello.manifest uuid


hello: hello.c
	$(CC) -Os -o$@ $<


hello.manifest: hello.manifest.template
	graphene-manifest $< > $@


premain-libos:
	wget https://github.com/edgelesssys/marblerun/releases/latest/download/premain-libos
	chmod u+x premain-libos

run:
	graphene-sgx hello
