include makefiles/common.mk
include makefiles/osx.mk
include makefiles/windows.mk

clean:
	rm -rf ${OUTPUT}

${OUTPUT}/:
	@mkdir $@

