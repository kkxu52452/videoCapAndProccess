# Compiler
OW_USER?=openwhisk
OW_COMPILER?=$(OW_USER)/actionloop-golang-v1.11

SRCS=main.go

# Output
NAME=hello-stranger
BINZIP=$(NAME)-bin.zip
SRCZIP=$(NAME)-src.zip


binzip:
    docker run -i $(OW_COMPILER) -compile main <$(SRCZIP) >$(BINZIP)
	echo "Make binary zip done!"

$(SRCZIP): $(SRCS)
	zip $(SRCZIP) -qr *
