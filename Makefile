include makefiles/common.mk
include makefiles/windows.mk

clean:
	rm -rf ${OUTPUT}

${OUTPUT}/:
	@mkdir $@

msi: ${WINDOWS}/${PROJECT}_${TAG}.msi
